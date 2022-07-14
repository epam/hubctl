// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

const stackInstancesResource = "hub/api/v1/instances"

var (
	stackInstancesCache = make(map[string]*StackInstance)
	queryMarkers        = []string{"<", ">", "="}
)

func StackInstances(selector, environmentSelector string, showSecrets, showLogs, showBackups, jsonFormat bool) {
	environmentId := ""
	if environmentSelector != "" {
		environment, err := environmentBy(environmentSelector)
		if err != nil {
			log.Fatalf("Unable to query for Environment: %v", err)
		}
		if environment == nil {
			log.Fatalf("No Environment `%s` found", environmentSelector)
		}
		environmentId = environment.Id
	}
	instances, err := stackInstancesBy(selector, environmentId)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}
	if len(instances) == 0 {
		if jsonFormat {
			log.Print("No Stack Instances")
		} else {
			fmt.Print("No Stack Instances\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(instances) == 1 {
				toMarshal = &instances[0]
			} else {
				toMarshal = instances
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Stack Instances:\n")
			errors := make([]error, 0)
			for _, instance := range instances {
				errors = formatStackInstanceEntity(&instance, showSecrets, showLogs, showBackups, errors)
			}
			if len(errors) > 0 {
				fmt.Print("Errors encountered:\n")
				for _, err := range errors {
					fmt.Printf("\t%v\n", err)
				}
			}
		}
	}
}

func formatStackInstanceEntity(instance *StackInstance, showSecrets, showLogs, showBackups bool, errors []error) []error {
	title := fmt.Sprintf("%s / %s [%s]", instance.Name, instance.Domain, instance.Id)
	if instance.Description != "" {
		title = fmt.Sprintf("%s - %s", title, instance.Description)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(instance.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(instance.Tags, ", "))
	}
	if instance.Environment.Name != "" {
		fmt.Printf("\t\tEnvironment: %s\n", formatEnvironmentRef(&instance.Environment))
	}
	if instance.Platform != nil && instance.Platform.Name != "" {
		fmt.Printf("\t\tPlatform: %s\n", formatPlatformRef(instance.Platform))
	}
	if instance.Stack.Name != "" {
		fmt.Printf("\t\tStack: %s\n", formatStackRef(&instance.Stack))
	}
	if instance.Template.Name != "" {
		fmt.Printf("\t\tTemplate: %s\n", formatTemplateRef(&instance.Template))
	}
	if len(instance.ComponentsEnabled) > 0 {
		fmt.Printf("\t\tComponents: %s\n", strings.Join(instance.ComponentsEnabled, ", "))
	}
	if len(instance.Verbs) > 0 {
		fmt.Printf("\t\tVerbs: %s\n", strings.Join(instance.Verbs, ", "))
	}
	if g := instance.GitRemote; g.Public != "" {
		templateRef := ""
		if g.Template != nil && g.Template.Ref != "" {
			templateRef = fmt.Sprintf("\n\t\t\tRef: %s", g.Template.Ref)
		}
		k8sRef := ""
		if g.K8s != nil && g.K8s.Ref != "" {
			k8sRef = fmt.Sprintf("\n\t\t\tstack-k8s-aws ref: %s", g.K8s.Ref)
		}
		fmt.Printf("\t\tGit: %s%s%s\n", g.Public, templateRef, k8sRef)
	}
	if instance.Status.Status != "" {
		fmt.Printf("\t\tStatus: %s\n", instance.Status.Status)
	}
	if len(instance.StateFiles) > 0 {
		fmt.Printf("\t\tState files:\n\t\t\t%s\n", strings.Join(instance.StateFiles, "\n\t\t\t"))
	}
	if len(instance.Provides) > 0 {
		formatted := formatStackProvides(instance.Provides, "\t\t\t")
		fmt.Printf("\t\tProvides:\n%s\n", formatted)
	}
	resource := fmt.Sprintf("%s/%s", stackInstancesResource, instance.Id)
	if len(instance.Outputs) > 0 {
		formatted, errs := formatStackOutputs(resource, instance.Outputs, showSecrets)
		fmt.Printf("\t\tOutputs:\n%s", formatted)
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}
	if len(instance.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
		for _, param := range sortParameters(instance.Parameters) {
			formatted, err := formatParameter(resource, param, showSecrets)
			fmt.Printf("\t\t%s\n", formatted)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if instance.Status.Template != nil && instance.Status.Template.Commit != "" {
		t := instance.Status.Template
		commit := t.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("\t\tTemplate deployed: %s %s %s %v %s\n", commit, t.Ref, t.Author, t.Date, t.Subject)
	}
	if instance.Status.K8s != nil && instance.Status.K8s.Commit != "" {
		t := instance.Status.K8s
		commit := t.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("\t\tKubernetes deployed: %s %s %s %v %s\n", commit, t.Ref, t.Author, t.Date, t.Subject)
	}
	if len(instance.Status.Components) > 0 {
		fmt.Print("\t\tComponents Status:\n")
		for _, comp := range instance.Status.Components {
			fmt.Print(formatComponentStatus(comp))
		}
	}
	if len(instance.InflightOperations) > 0 {
		fmt.Print("\t\tInflight Operations:\n")
		for _, op := range instance.InflightOperations {
			fmt.Print(formatInflightOperation(op, showLogs))
		}
	}
	if showBackups {
		backups, err := backupsByInstanceId(instance.Id)
		if err != nil {
			errors = append(errors, err)
		}
		if len(backups) > 0 {
			fmt.Print("\tBackups:\n")
			errs := make([]error, 0)
			for _, backup := range backups {
				errs = formatBackupEntity(&backup, showLogs, errors)
			}
			if len(errs) > 0 {
				errors = append(errors, errs...)
			}
		}
		if instance.Platform == nil {
			backups, err = backupsByPlatformId(instance.Id)
			if err != nil {
				errors = append(errors, err)
			}
			if len(backups) > 0 {
				fmt.Print("\tOverlay backups:\n")
				errs := make([]error, 0)
				for _, backup := range backups {
					errs = formatBackupEntity(&backup, showLogs, errors)
				}
				if len(errs) > 0 {
					errors = append(errors, errs...)
				}
			}
		}
	}
	return errors
}

func formatStackInstance(instance *StackInstance) {
	errors := formatStackInstanceEntity(instance, false, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func cachedStackInstanceBy(selector string) (*StackInstance, error) {
	instance, cached := stackInstancesCache[selector]
	if !cached {
		var err error
		instance, err = stackInstanceBy(selector)
		if err != nil {
			return instance, err
		}
		stackInstancesCache[selector] = instance
	}
	return instance, nil
}

func stackInstanceBy(selector string) (*StackInstance, error) {
	if !util.IsUint(selector) {
		return stackInstanceByDomain(selector)
	}
	return stackInstanceById(selector)
}

func maybeQuery(selector string) bool {
	for _, marker := range queryMarkers {
		if strings.Contains(selector, marker) {
			return true
		}
	}
	return false
}

func stackInstancesBy(selector, environmentId string) ([]StackInstance, error) {
	if !util.IsUint(selector) {
		if maybeQuery(selector) {
			return stackInstancesByQuery(selector, environmentId)
		}
		return stackInstancesByDomain(selector, environmentId)
	}
	instance, err := stackInstanceById(selector)
	if err != nil {
		return nil, err
	}
	if instance != nil {
		return []StackInstance{*instance}, nil
	}
	return nil, nil
}

func stackInstanceById(id string) (*StackInstance, error) {
	path := fmt.Sprintf("%s/%s", stackInstancesResource, url.PathEscape(id))
	var jsResp StackInstance
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Stack Instances: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Stack Instances, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func stackInstanceByDomain(domain string) (*StackInstance, error) {
	instances, err := stackInstancesByDomain(domain, "")
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Stack Instance `%s`: %v", domain, err)
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("No Stack Instance `%s` found", domain)
	}
	if len(instances) > 1 {
		return nil, fmt.Errorf("More than one Stack Instance returned by domain `%s`", domain)
	}
	instance := instances[0]
	return &instance, nil
}

func stackInstancesByQuery(query, environmentId string) ([]StackInstance, error) {
	return stackInstancesByField(query, environmentId, "query")
}

func stackInstancesByDomain(domain, environmentId string) ([]StackInstance, error) {
	return stackInstancesByField(domain, environmentId, "domain")
}

func stackInstancesByField(selector, environmentId, field string) ([]StackInstance, error) {
	var args []string
	if environmentId != "" {
		args = append(args, fmt.Sprintf("environment=%s", url.QueryEscape(environmentId)))
	}
	if selector != "" {
		args = append(args, fmt.Sprintf("%s=%s", field, url.QueryEscape(selector)))
	}
	path := stackInstancesResource
	if len(args) > 0 {
		path = fmt.Sprintf("%s?%s", stackInstancesResource, strings.Join(args, "&"))
	}
	var jsResp []StackInstance
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Stack Instances: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Stack Instances, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func stackInstancesByPlatform(platformId string) ([]StackInstance, error) {
	path := fmt.Sprintf("%s/%s/overlays", stackInstancesResource, url.PathEscape(platformId))
	var jsResp []StackInstance
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Stack Instances: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Stack Instances, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatStackInstanceRef(ref *StackInstanceRef, resource string) (string, []error) {
	errors := make([]error, 0)
	parameters := ""
	if len(ref.Parameters) > 0 {
		formattedParameters := make([]string, 0, len(ref.Parameters))
		for _, param := range sortParameters(ref.Parameters) {
			formatted, err := formatParameter(resource, param, false)
			formattedParameters = append(formattedParameters, fmt.Sprintf("\n\t\t\t%s", formatted))
			if err != nil {
				errors = append(errors, err)
			}
		}
		parameters = fmt.Sprintf("\t\t\tParameters:%s", strings.Join(formattedParameters, ""))
	}
	stack := ""
	if ref.Stack.Name != "" {
		stack = fmt.Sprintf("\n\t\t\tStack: %s\n", formatStackRef(&ref.Stack))
	}
	platform := ""
	if ref.Platform.Id != "" {
		platform = fmt.Sprintf("\n\t\t\tPlatform: %s [%s]", ref.Platform.Domain, ref.Platform.Id)
	}
	return fmt.Sprintf("%s / %s [%s]%s%s%s", ref.Name, ref.Domain, ref.Id, platform, stack, parameters), errors
}

func formatPlatformRef(ref *PlatformRef) string {
	stateFiles := ""
	if len(ref.StateFiles) > 0 {
		stateFiles = fmt.Sprintf("\n\t\t\tState files:\n\t\t\t\t%s", strings.Join(ref.StateFiles, "\n\t\t\t\t"))
	}
	provides := ""
	if len(ref.Provides) > 0 {
		provides = fmt.Sprintf("\n\t\t\tProvides:\n%s", formatStackProvides(ref.Provides, "\t\t\t\t"))
	}
	return fmt.Sprintf("%s / %s [%s]%s%s", ref.Name, ref.Domain, ref.Id, stateFiles, provides)
}

func formatStackProvides(provides map[string][]string, indent string) string {
	str := make([]string, 0, len(provides))
	for _, k := range util.SortedKeys2(provides) {
		str = append(str, fmt.Sprintf("%s => %s", k, strings.Join(provides[k], ", ")))
	}
	return fmt.Sprintf("%s%s", indent, strings.Join(str, "\n"+indent))
}

func formatStackOutputs(resource string, outputs []Output, showSecrets bool) (string, []error) {
	ident := "\t\t"
	str := make([]string, 0, len(outputs))
	errors := make([]error, 0)
	for _, o := range outputs {
		brief := ""
		if o.Brief != "" {
			brief = fmt.Sprintf(" [%s]", o.Brief)
		}
		messenger := ""
		if o.Messenger != "" {
			messenger = fmt.Sprintf(" *%s*", o.Messenger)
		}
		comp := ""
		if o.Component != "" {
			comp = fmt.Sprintf("%s:", o.Component)
		}
		title := fmt.Sprintf("%7s %s%s:", o.Kind, comp, o.Name)
		formatted, err := formatParameterValue(resource, o.Kind, o.Value, showSecrets)
		if err != nil {
			errors = append(errors, err)
		}
		var entry string
		if strings.Contains(formatted, "\n") {
			maybeNl := "\n"
			if strings.HasSuffix(formatted, "\n") {
				maybeNl = ""
			}
			entry = fmt.Sprintf("%s ~~%s%s %s%s~~", title, brief, messenger, formatted, maybeNl)
		} else {
			entry = fmt.Sprintf("%s %s%s%s", title, formatted, brief, messenger)
		}
		str = append(str, entry)
	}
	return fmt.Sprintf("%s%s\n", ident, strings.Join(str, "\n"+ident)), errors
}

func formatComponentStatus(comp ComponentStatus) string {
	ident := "\t\t\t"
	origin := ""
	title := ""
	version := ""
	if meta := comp.Meta; meta != nil {
		if meta.Origin != "" && meta.Origin != comp.Name {
			origin = meta.Origin
		}
		if meta.Kind != "" && meta.Kind != meta.Origin {
			origin = fmt.Sprintf("%s/%s", origin, meta.Kind)
		}
		if meta.Title != "" {
			title = fmt.Sprintf(" - %s", meta.Title)
		}
		version = meta.Version
	}
	if version == "" && comp.Version != "" {
		version = comp.Version
	}
	var etc []string
	if origin != "" {
		etc = append(etc, origin)
	}
	if version != "" {
		etc = append(etc, version)
	}
	printEtc := ""
	if len(etc) > 0 {
		printEtc = fmt.Sprintf(" [%s]", strings.Join(etc, " "))
	}
	message := ""
	if comp.Message != "" {
		sep := " "
		if strings.Contains(comp.Message, "\n") {
			sep = "\n"
		}
		message = fmt.Sprintf(":%s%s", sep, comp.Message)
	}
	timestamps := ""
	if t := comp.Timestamps; t != nil {
		timestamps = fmt.Sprintf(" (start: %v; duration %s)",
			t.Start.Truncate(time.Second), t.End.Sub(t.Start).Truncate(time.Second).String())
	}
	str := fmt.Sprintf("%s%s%s%s - %s%s%s\n", ident, comp.Name, printEtc, title, comp.Status, timestamps, message)
	if len(comp.Outputs) > 0 {
		str = fmt.Sprintf("%s%s\t%s\n", str, ident, formatComponentOutputs(comp.Outputs, ident))
	}
	return str
}

func formatComponentOutputs(outputs []ComponentOutput, ident string) string {
	keys := make([]string, 0, len(outputs))
	values := make(map[string]ComponentOutput)
	for _, output := range outputs {
		keys = append(keys, output.Name)
		values[output.Name] = output
	}
	sort.Strings(keys)
	str := make([]string, 0, len(outputs))
	for _, name := range keys {
		output := values[name]
		brief := ""
		if output.Brief != "" {
			brief = fmt.Sprintf(" [%s]", output.Brief)
		}
		str = append(str, fmt.Sprintf("%s%s: %+v", name, brief, output.Value))
	}
	return strings.Join(str, "\n\t"+ident)
}

func formatInflightOperation(op InflightOperation, showLogs bool) string {
	ident := "\t\t\t"
	logs := ""
	if showLogs && op.Logs != "" {
		logs = fmt.Sprintf("%sLogs:\n%s\t%s\n",
			ident, ident, strings.Join(strings.Split(op.Logs, "\n"), "\n"+ident+"\t"))
	}
	initiator := ""
	if op.Initiator != "" {
		initiator = fmt.Sprintf(" by %s", op.Initiator)
	}
	options := ""
	if len(op.Options) > 0 {
		options = fmt.Sprintf("%s\tOptions: %v\n", ident, op.Options)
	}
	location := ""
	if op.Location != "" {
		location = fmt.Sprintf("%s\tLocation: %v\n", ident, op.Location)
	}
	platform := ""
	if op.PlatformDomain != "" {
		platform = fmt.Sprintf("%s\tPlatform: %v\n", ident, op.PlatformDomain)
	}
	description := ""
	if op.Description != "" {
		description = fmt.Sprintf(" (%s)", op.Description)
	}
	phases := ""
	if len(op.Phases) > 0 {
		phases = fmt.Sprintf("%s\tPhases:\n%s\t\t%s\n", ident, ident, formatLifecyclePhases(op.Phases, ident+"\t"))
	}
	return fmt.Sprintf("%sOperation: %s - %s %v%s%s %s\n%s%s%s%s%s",
		ident, op.Operation, op.Status, op.Timestamp.Truncate(time.Second), initiator, description, op.Id,
		platform, options, location, phases, logs)
}

func formatLifecyclePhases(phases []LifecyclePhase, ident string) string {
	str := make([]string, 0, len(phases))
	for _, phase := range phases {
		str = append(str, fmt.Sprintf("%s - %s", phase.Phase, phase.Status))
	}
	return strings.Join(str, "\n"+ident+"\t")
}

func CreateStackInstance(req StackInstanceRequest) {
	stackInstance, err := createStackInstance(req)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Stack Instance: %v", err)
	}
	formatStackInstance(stackInstance)
}

func createStackInstance(req StackInstanceRequest) (*StackInstance, error) {
	if req.Template != "" && !util.IsUint(req.Template) {
		template, err := templateByName(req.Template)
		if err != nil {
			return nil, err
		}
		req.Template = template.Id
	}
	if req.Environment != "" && !util.IsUint(req.Environment) {
		environment, err := environmentByName(req.Environment)
		if err != nil {
			return nil, err
		}
		req.Environment = environment.Id
	}
	if req.Platform != "" && !util.IsUint(req.Platform) {
		platform, err := stackInstanceByDomain(req.Platform)
		if err != nil {
			return nil, err
		}
		req.Platform = platform.Id
	}
	var jsResp StackInstance
	code, err := post(hubApi(), stackInstancesResource, &req, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Stack Instance, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func RawCreateStackInstance(body io.Reader) {
	stackInstance, err := rawCreateStackInstance(body)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Stack Instance: %v", err)
	}
	formatStackInstance(stackInstance)
}

func rawCreateStackInstance(body io.Reader) (*StackInstance, error) {
	var jsResp StackInstance
	code, err := post2(hubApi(), stackInstancesResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Stack Instance, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func DeployStackInstance(selector string, components []string, waitAndTailDeployLogs, dryRun bool) {
	_, err := commandStackInstance(selector, "deploy", StackInstanceLifecycleRequest{components}, waitAndTailDeployLogs, dryRun)
	if err != nil {
		log.Fatalf("Unable to deploy SuperHub Stack Instance: %v", err)
	}
}

func UndeployStackInstance(selector string, components []string, waitAndTailDeployLogs bool) {
	_, err := commandStackInstance(selector, "undeploy", StackInstanceLifecycleRequest{components}, waitAndTailDeployLogs, false)
	if err != nil {
		log.Fatalf("Unable to undeploy SuperHub Stack Instance: %v", err)
	}
}

func BackupStackInstance(selector, name string, components []string, waitAndTailDeployLogs bool) {
	resp, err := commandStackInstance(selector, "backup", &BackupRequest{Name: name, Components: components}, false, false)
	if err != nil {
		log.Fatalf("Unable to backup SuperHub Stack Instance: %v", err)
	}
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		code := Logs([]string{"backup/" + resp.Id}, true)
		showBackup(resp.Id)
		os.Exit(code)
	}
	showBackup(resp.Id)
}

func commandStackInstance(selector, verb string, req interface{}, waitAndTailDeployLogs, dryRun bool) (*StackInstanceLifecycleResponse, error) {
	instance, err := cachedStackInstanceBy(selector)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, error404
	}
	maybeDryRun := ""
	if dryRun {
		maybeDryRun = "?dryRun=true"
	}
	var jsResp StackInstanceLifecycleResponse
	path := fmt.Sprintf("%s/%s/%s%s", stackInstancesResource, url.PathEscape(instance.Id), verb, maybeDryRun)
	code, err := post(hubApi(), path, req, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 202 && code != 204 {
		return nil, fmt.Errorf("Got %d HTTP in response to %s SuperHub Stack Instance, expected [200, 202, 204] HTTP",
			code, verb)
	}
	if config.Verbose {
		log.Printf("Instance %s automation task id: %s", verb, jsResp.JobId)
	}
	if waitAndTailDeployLogs && !dryRun {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{instance.Id}, true))
	}
	return &jsResp, nil
}

func DeleteStackInstance(selector string) {
	err := deleteStackInstance(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Stack Instance: %v", err)
	}
}

func deleteStackInstance(selector string) error {
	instance, err := cachedStackInstanceBy(selector)
	id := ""
	if err != nil {
		str := err.Error()
		if util.IsUint(selector) &&
			(strings.Contains(str, "json: cannot unmarshal") || strings.Contains(str, "cannot parse") || config.Force) {
			util.Warn("%v", err)
			id = selector
		} else {
			return err
		}
	} else if instance == nil {
		return error404
	} else {
		id = instance.Id
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", stackInstancesResource, url.PathEscape(id), force)
	code, err := delete(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Stack Instance, expected [202, 204] HTTP", code)
	}
	return nil
}

func KubeconfigStackInstance(selector, filename string) {
	err := kubeconfigStackInstance(selector, filename)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Stack Instance Kubeconfig: %v", err)
	}
}

func kubeconfigStackInstance(selector, filename string) error {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return err
	}
	if instance == nil {
		return error404
	}
	path := fmt.Sprintf("%s/%s/kubeconfig", stackInstancesResource, url.PathEscape(instance.Id))
	code, body, err := get2(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("Got %d HTTP fetching SuperHub Stack Instance Kubeconfig, expected 200 HTTP", code)
	}
	if len(body) == 0 {
		return fmt.Errorf("Got empty SuperHub Stack Instance Kubeconfig")
	}

	if filename == "" {
		filename = fmt.Sprintf("kubeconfig.%s.yaml", instance.Domain)
	}
	var file io.WriteCloser
	if filename == "-" {
		file = os.Stdout
	} else {
		info, _ := os.Stat(filename)
		if info != nil {
			if info.IsDir() {
				filename = fmt.Sprintf("%s/kubeconfig", filename)
			} else {
				if !config.Force {
					log.Fatalf("Kubeconfig `%s` exists, use --force / -f to overwrite", filename)
				}
			}
		}
		var err error
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("Unable to create %s: %v", filename, err)
		}
		defer file.Close()
	}
	written, err := file.Write(body)
	if written != len(body) {
		return fmt.Errorf("Unable to write %s: %v", filename, err)
	}
	if config.Verbose && filename != "-" {
		log.Printf("Wrote %s", filename)
	}

	return nil
}

func LogsStackInstance(selector, operationId, filename string) {
	err := logsStackInstance(selector, operationId, filename)
	if err != nil {
		log.Fatalf("Unable to download SuperHub Stack Instance log: %v", err)
	}
}

func logsStackInstance(selector, operationId, filename string) error {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return err
	}
	if instance == nil {
		return error404
	}
	ops := instance.InflightOperations
	if len(ops) == 0 {
		return errors.New("No inflight operations")
	}
	var op InflightOperation
	if operationId == "" {
		op = ops[len(ops)-1]
	} else {
		for i := range ops {
			if operationId == ops[i].Id {
				op = ops[i]
			}
		}
	}
	if op.Id == "" {
		return errors.New("No inflight operation found")
	}
	if op.Location == "" {
		return errors.New("Inflight operation has no location")
	}
	path := fmt.Sprintf("%s/%s?log=true", tasksResource, op.Location)
	code, body, err := get2(hubApi(), path) // TODO stream log into file
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("Got %d HTTP fetching SuperHub Stack Instance log, expected 200 HTTP", code)
	}
	if len(body) == 0 {
		return fmt.Errorf("Got empty SuperHub Stack Instance log")
	}

	if filename == "" {
		filename = fmt.Sprintf("logs.%s.%s.txt", instance.Domain, op.Id)
	}
	var file io.WriteCloser
	if filename == "-" {
		file = os.Stdout
	} else {
		info, _ := os.Stat(filename)
		if info != nil {
			if info.IsDir() {
				filename = fmt.Sprintf("%s/%s", filename, op.Id)
			} else {
				if !config.Force {
					log.Fatalf("Log `%s` exists, use --force / -f to overwrite", filename)
				}
			}
		}
		var err error
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("Unable to create %s: %v", filename, err)
		}
		defer file.Close()
	}
	written, err := file.Write(body)
	if written != len(body) {
		return fmt.Errorf("Unable to write %s: %v", filename, err)
	}
	if config.Verbose && filename != "-" {
		log.Printf("Wrote %s", filename)
	}

	return nil
}

func PatchStackInstanceForCmd(selector string, change StackInstancePatch, replace bool) {
	stackInstance, err := PatchStackInstance(selector, change, replace)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Stack Instance: %v", err)
	}
	formatStackInstance(stackInstance)
}

func PatchStackInstance(selector string, change StackInstancePatch, replace bool) (*StackInstance, error) {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, error404
	}
	// reset `gitRemote` as we may have unmarshalled empty struct due to presence of `public` field
	// which is not allowed in patch
	if change.GitRemote != nil && change.GitRemote.Template == nil && change.GitRemote.K8s == nil {
		change.GitRemote = nil
	}
	maybeReplace := ""
	if replace {
		maybeReplace = "?replace=1"
	}
	path := fmt.Sprintf("%s/%s%s", stackInstancesResource, url.PathEscape(instance.Id), maybeReplace)
	var jsResp StackInstance
	code, err := patch(hubApi(), path, &change, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Stack Instance, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func RawPatchStackInstance(selector string, body io.Reader, replace bool) {
	stackInstance, err := rawPatchStackInstance(selector, body, replace)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Stack Instance: %v", err)
	}
	formatStackInstance(stackInstance)
}

func rawPatchStackInstance(selector string, body io.Reader, replace bool) (*StackInstance, error) {
	instance, err := stackInstanceBy(selector)
	if err != nil && !config.Force {
		return nil, err
	}
	if instance == nil && !config.Force {
		return nil, error404
	}
	instanceId := ""
	if instance == nil {
		if util.IsUint(selector) {
			instanceId = selector
		} else {
			return nil, errors.New("Specify instance by Id")
		}
	} else {
		instanceId = instance.Id
	}
	maybeReplace := ""
	if replace {
		maybeReplace = "?replace=1"
	}
	path := fmt.Sprintf("%s/%s%s", stackInstancesResource, url.PathEscape(instanceId), maybeReplace)
	var jsResp StackInstance
	code, err := patch2(hubApi(), path, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Stack Instance, expected 200 HTTP", code)
	}
	return &jsResp, nil
}
