package api

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"hub/config"
	"hub/util"
)

const stackInstancesResource = "hub/api/v1/instances"

var stackInstancesCache = make(map[string]*StackInstance)

func StackInstances(selector string, showSecrets, showLogs bool) {
	instances, err := stackInstancesBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}
	if len(instances) == 0 {
		fmt.Print("No Stack Instances\n")
	} else {
		fmt.Print("Stack Instances:\n")
		errors := make([]error, 0)
		for _, instance := range instances {
			errors = formatStackInstanceEntity(&instance, showSecrets, showLogs, errors)
		}
		if len(errors) > 0 {
			fmt.Print("Errors encountered:\n")
			for _, err := range errors {
				fmt.Printf("\t%v\n", err)
			}
		}
	}
}

func formatStackInstanceEntity(instance *StackInstance, showSecrets, showLogs bool, errors []error) []error {
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
	if instance.Platform.Name != "" {
		fmt.Printf("\t\tPlatform: %s\n", formatPlatformRef(&instance.Platform))
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
	if instance.GitRemote.Public != "" {
		g := instance.GitRemote
		templateRef := ""
		if g.Template.Ref != "" {
			templateRef = fmt.Sprintf("\n\t\t\tRef: %s", g.Template.Ref)
		}
		k8sRef := ""
		if g.K8s.Ref != "" {
			k8sRef = fmt.Sprintf("\n\t\t\tstack-k8s-aws ref: %s", g.K8s.Ref)
		}
		fmt.Printf("\t\tGit: %s%s%s\n", g.Public, templateRef, k8sRef)
	}
	if len(instance.StateFiles) > 0 {
		fmt.Printf("\t\tState files:\n\t\t\t%s\n", strings.Join(instance.StateFiles, "\n\t\t\t"))
	}
	if len(instance.Provides) > 0 {
		formatted := formatStackProvides(instance.Provides)
		fmt.Printf("\t\tProvides:\n%s", formatted)
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
	}
	for _, param := range sortParameters(instance.Parameters) {
		formatted, err := formatParameter(resource, param, showSecrets)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if instance.Status.Status != "" {
		fmt.Printf("\t\tStatus: %s\n", instance.Status.Status)
	}
	if instance.Status.Template != nil && instance.Status.Template.Commit != "" {
		t := instance.Status.Template
		commit := t.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("\t\tTemplate deployed: %s %s %s %s %s\n", commit, t.Ref, t.Author, t.Date, t.Subject)
	}
	if instance.Status.K8s != nil && instance.Status.K8s.Commit != "" {
		t := instance.Status.K8s
		commit := t.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		fmt.Printf("\t\tKubernetes deployed: %s %s %s %s %s\n", commit, t.Ref, t.Author, t.Date, t.Subject)
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
	return errors
}

func cachedStackInstanceBy(selector string) (*StackInstance, error) {
	instance, cached := stackInstancesCache[selector]
	if !cached {
		var err error
		instance, err = stackInstanceBy(selector)
		if err != nil {
			return nil, err
		}
		stackInstancesCache[selector] = instance
	}
	return instance, nil
}

func stackInstanceBy(selector string) (*StackInstance, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return stackInstanceByDomain(selector)
	}
	return stackInstanceById(selector)
}

func stackInstancesBy(selector string) ([]StackInstance, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return stackInstancesByDomain(selector)
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
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Stack Instances: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Stack Instances, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func stackInstanceByDomain(domain string) (*StackInstance, error) {
	instances, err := stackInstancesByDomain(domain)
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

func stackInstancesByDomain(domain string) ([]StackInstance, error) {
	path := stackInstancesResource
	if domain != "" {
		path += "?domain=" + url.QueryEscape(domain)
	}
	var jsResp []StackInstance
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Stack Instances: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Stack Instances, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatPlatformRef(ref *PlatformRef) string {
	return fmt.Sprintf("%s / %s [%s]", ref.Name, ref.Domain, ref.Id)
}

func formatStackProvides(provides map[string][]string) string {
	ident := "\t\t\t"
	str := make([]string, 0, len(provides))
	for _, k := range util.SortedKeys2(provides) {
		str = append(str, fmt.Sprintf("%s => %s", k, strings.Join(provides[k], ", ")))
	}
	return fmt.Sprintf("%s%s\n", ident, strings.Join(str, "\n"+ident))
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
	message := ""
	if comp.Message != "" {
		message = fmt.Sprintf(": %s", comp.Message)
	}
	str := fmt.Sprintf("%s%s - %s%s\n", ident, comp.Name, comp.Status, message)
	if len(comp.Outputs) > 0 {
		str = fmt.Sprintf("%s%s\t%s\n", str, ident, formatComponentOutputs(comp.Outputs, ident))
	}
	return str
}

func formatComponentOutputs(outputs map[string]string, ident string) string {
	keys := make([]string, 0, len(outputs))
	for key := range outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	str := make([]string, 0, len(outputs))
	for _, name := range keys {
		str = append(str, fmt.Sprintf("%s: %s", name, outputs[name]))
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
		options = fmt.Sprintf("%sOptions: %v\n", ident, op.Options)
	}
	description := ""
	if op.Description != "" {
		description = fmt.Sprintf(" (%s)", op.Description)
	}
	phases := ""
	if len(op.Phases) > 0 {
		phases = fmt.Sprintf("%sPhases:\n%s\t%s\n", ident, ident, formatLifecyclePhases(op.Phases, ident))
	}
	return fmt.Sprintf("%sOperation: %s - %s %v%s%s %s\n%s%s%s",
		ident, op.Operation, op.Status, op.Timestamp, initiator, description, op.Id, options, phases, logs)
}

func formatLifecyclePhases(phases []LifecyclePhase, ident string) string {
	str := make([]string, 0, len(phases))
	for _, phase := range phases {
		str = append(str, fmt.Sprintf("%s - %s", phase.Phase, phase.Status))
	}
	return strings.Join(str, "\n"+ident+"\t")
}

func CreateStackInstance(body io.Reader) {
	stackInstance, err := createStackInstance(body)
	if err != nil {
		log.Fatalf("Unable to create Hub Service Stack Instance: %v", err)
	}
	errors := formatStackInstanceEntity(stackInstance, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func createStackInstance(body io.Reader) (*StackInstance, error) {
	var jsResp StackInstance
	code, err := post2(hubApi, stackInstancesResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating Hub Service Stack Instance, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func DeployStackInstance(selector string, waitAndTailDeployLogs, dryRun bool) {
	err := commandStackInstance(selector, "deploy", waitAndTailDeployLogs, dryRun)
	if err != nil {
		log.Fatalf("Unable to deploy Hub Service Stack Instance: %v", err)
	}
}

func UndeployStackInstance(selector string, waitAndTailDeployLogs bool) {
	err := commandStackInstance(selector, "undeploy", waitAndTailDeployLogs, false)
	if err != nil {
		log.Fatalf("Unable to undeploy Hub Service Stack Instance: %v", err)
	}
}

func commandStackInstance(selector, verb string, waitAndTailDeployLogs, dryRun bool) error {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return err
	}
	if instance == nil {
		return error404
	}
	maybeDryRun := ""
	if dryRun {
		maybeDryRun = "?dryRun=1"
	}
	var jsResp StackInstanceDeployResponse
	path := fmt.Sprintf("%s/%s/%s%s", stackInstancesResource, url.PathEscape(instance.Id), verb, maybeDryRun)
	code, err := post2(hubApi, path, nil, &jsResp)
	if err != nil {
		return err
	}
	if code != 200 && code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP in response to %s Hub Service Stack Instance, expected [200, 202, 204] HTTP",
			code, verb)
	}
	if config.Verbose {
		log.Printf("Instance %s automation task id: %s", verb, jsResp.JobId)
	}
	if waitAndTailDeployLogs {
		// TODO wait for stackInstance status update and exit on success of failure
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		Logs([]string{instance.Domain})
	}
	return nil
}

func DeleteStackInstance(selector string) {
	err := deleteStackInstance(selector)
	if err != nil {
		log.Fatalf("Unable to delete Hub Service Stack Instance: %v", err)
	}
}

func deleteStackInstance(selector string) error {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return err
	}
	if instance == nil {
		return error404
	}
	path := fmt.Sprintf("%s/%s", stackInstancesResource, url.PathEscape(instance.Id))
	code, err := delete(hubApi, path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting Hub Service Stack Instance, expected [202, 204] HTTP", code)
	}
	return nil
}

func KubeconfigStackInstance(selector, filename string) {
	err := kubeconfigStackInstance(selector, filename)
	if err != nil {
		log.Fatalf("Unable to create Hub Service Stack Instance Kubeconfig: %v", err)
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
	path := fmt.Sprintf("%s/%s/config", stackInstancesResource, url.PathEscape(instance.Id))
	code, err, body := get2(hubApi, path)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("Got %d HTTP fetching Hub Service Stack Instance Kubeconfig, expected 200 HTTP", code)
	}
	if len(body) == 0 {
		return fmt.Errorf("Got empty Hub Service Stack Instance Kubeconfig")
	}

	if filename == "" {
		filename = fmt.Sprintf("kubeconfig-%s.yaml", instance.Domain)
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

func PatchStackInstance(selector string, change StackInstancePatch) (*StackInstance, error) {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s?replace=1", stackInstancesResource, url.PathEscape(instance.Id))
	var jsResp StackInstance
	code, err := patch(hubApi, path, &change, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching Hub Service Stack Instance, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func RawPatchStackInstance(selector string, body io.Reader, replace bool) {
	stackInstance, err := rawPatchStackInstance(selector, body, replace)
	if err != nil {
		log.Fatalf("Unable to patch Hub Service Stack Instance: %v", err)
	}
	errors := formatStackInstanceEntity(stackInstance, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func rawPatchStackInstance(selector string, body io.Reader, replace bool) (*StackInstance, error) {
	instance, err := stackInstanceBy(selector)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, error404
	}
	maybeReplace := ""
	if replace {
		maybeReplace = "?replace=1"
	}
	path := fmt.Sprintf("%s/%s%s", stackInstancesResource, url.PathEscape(instance.Id), maybeReplace)
	var jsResp StackInstance
	code, err := patch2(hubApi, path, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching Hub Service Stack Instance, expected 200 HTTP", code)
	}
	return &jsResp, nil
}
