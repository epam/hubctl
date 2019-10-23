package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"hub/config"
	"hub/util"
)

var (
	workerpoolStacks = []string{"k8s-worker-node-pool:1", "eks-worker-node-pool:1", "gke-worker-node-pool:1"}
)

func Workerpools(selector string, showSecrets, showLogs, jsonFormat bool) {
	instances, err := stackInstancesBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}
	workerpools := make([]*StackInstance, 0, len(instances))
	errors := make([]error, 0)
	added := make(map[string]struct{})
	for i, instance := range instances {
		if instance.Platform != nil {
			if util.Contains(workerpoolStacks, instance.Stack.Id) {
				if _, seen := added[instance.Id]; !seen {
					workerpools = append(workerpools, &instances[i])
					added[instance.Id] = struct{}{}
				}
			}
		} else if selector != "" || len(workerpools) == 0 {
			overlays, err := stackInstancesByPlatform(instance.Id)
			if err != nil {
				errors = append(errors, err)
			}
			for j, overlay := range overlays {
				if util.Contains(workerpoolStacks, overlay.Stack.Id) {
					if _, seen := added[overlay.Id]; !seen {
						workerpools = append(workerpools, &overlays[j])
						added[overlay.Id] = struct{}{}
					}
				}
			}
		}
	}
	if len(errors) > 0 && (len(workerpools) == 0 || jsonFormat) {
		fmt.Print("Errors encountered:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
	if len(workerpools) == 0 {
		if jsonFormat {
			log.Print("No Worker Pools")
		} else {
			fmt.Print("No Worker Pools\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(workerpools) == 1 {
				toMarshal = workerpools[0]
			} else {
				toMarshal = workerpools
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Worker Pools:\n")
			for _, instance := range workerpools {
				errors = formatStackInstanceEntity(instance, showSecrets, showLogs, false, errors)
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

func CreateWorkerpool(selector, name, instanceType string, count, maxCount int,
	spotPrice float32, preemptibleVMs, autoscale bool, volumeSize int,
	waitAndTailDeployLogs, dryRun bool) {

	instance, err := stackInstanceBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}

	kind := "aws"
	if strings.HasPrefix(instance.Stack.Id, "gke") {
		kind = "gcp"
	}
	parameters := []Parameter{
		{Name: "component.k8s-worker-nodes.size", Value: instanceType},
		{Name: "component.k8s-worker-nodes.count", Value: count},
	}
	if maxCount > 0 {
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.maxCount", Value: maxCount})
	}
	if volumeSize > 0 {
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.volume.size", Value: volumeSize})
	}
	switch kind {
	case "aws":
		price := ""
		if spotPrice > 0 {
			price = fmt.Sprintf("%.2f", spotPrice) // TODO what if price is less than a cent?
		}
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.spotPrice", Value: price},
			Parameter{Name: "component.k8s-worker-nodes.autoscale.enabled", Value: autoscale})

	case "gcp":
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.preemptible", Value: preemptibleVMs})
	}
	req := &WorkerpoolRequest{
		Name:       name,
		Kind:       kind + "-worker-pool", // TODO API switching validation schema on input field
		Parameters: parameters,
	}
	maybeDryRun := ""
	if dryRun {
		maybeDryRun = "?dryRun=true"
	}
	path := fmt.Sprintf("%s/%s/workerpools%s", stackInstancesResource, url.PathEscape(instance.Id), maybeDryRun)
	var jsResp WorkerpoolLifecycleResponse
	code, err := post(hubApi, path, req, &jsResp)
	if err != nil {
		log.Fatalf("Error creating SuperHub `%s` Workerpool `%s`: %v",
			instance.Domain, name, err)
	}
	if code != 201 {
		log.Fatalf("Got %d HTTP creating SuperHub `%s` Workerpool `%s`, expected 201 HTTP",
			code, instance.Domain, name)
	}
	formatStackInstance(&jsResp.Instance)
	if waitAndTailDeployLogs && !dryRun {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"stackInstance/" + jsResp.Instance.Id}, true))
	}
}

func VerifyWorkerpool(selector string) {
	instance, err := cachedStackInstanceBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}
	if !util.Contains(workerpoolStacks, instance.Stack.Id) || instance.Platform == nil {
		log.Fatalf("Instance `%s` [%d] is not a workerpool", instance.Domain, instance.Id)
	}
}

func ScaleWorkerpool(selector, instanceType string, count, maxCount int, waitAndTailDeployLogs, dryRun bool) {
	VerifyWorkerpool(selector)
	instance, err := cachedStackInstanceBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack Instance(s): %v", err)
	}
	kind := "aws"
	if strings.HasPrefix(instance.Stack.Id, "gke") {
		kind = "gcp"
	}
	parameters := []Parameter{
		{Name: "component.k8s-worker-nodes.count", Value: count},
	}
	if instanceType != "" {
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.size", Value: instanceType})
	}
	if maxCount > 0 {
		parameters = append(parameters,
			Parameter{Name: "component.k8s-worker-nodes.maxCount", Value: maxCount})
	}
	req := &WorkerpoolPatch{
		Kind:       kind + "-worker-pool", // TODO API switching validation schema on input field
		Parameters: parameters,
	}
	maybeDryRun := ""
	if dryRun {
		maybeDryRun = "?dryRun=true"
	}
	path := fmt.Sprintf("%s/%s/workerpools/%s%s", stackInstancesResource,
		url.PathEscape(instance.Platform.Id), url.PathEscape(instance.Id), maybeDryRun)
	var jsResp WorkerpoolLifecycleResponse
	code, err := patch(hubApi, path, req, &jsResp)
	if err != nil {
		log.Fatalf("Error scaling SuperHub `%s` Workerpool `%s`: %v",
			instance.Platform.Domain, instance.Name, err)
	}
	if code != 202 {
		log.Fatalf("Got %d HTTP scaling SuperHub `%s` Workerpool `%s`, expected 202 HTTP",
			code, instance.Platform.Domain, instance.Name)
	}
	formatStackInstance(&jsResp.Instance)
	if waitAndTailDeployLogs && !dryRun {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"stackInstance/" + jsResp.Id}, true))
	}
}

func DeployWorkerpool(selector string, waitAndTailDeployLogs, dryRun bool) {
	VerifyWorkerpool(selector)
	_, err := commandStackInstance(selector, "deploy", nil, waitAndTailDeployLogs, dryRun)
	if err != nil {
		log.Fatalf("Unable to deploy SuperHub Workerpool: %v", err)
	}
}

func UndeployWorkerpool(selector string, useWorkerpoolApi, waitAndTailDeployLogs bool) {
	VerifyWorkerpool(selector)
	if useWorkerpoolApi {
		// use workerpool undeploy API (DELETE)
		instance, err := cachedStackInstanceBy(selector)
		if err != nil {
			log.Fatalf("Unable to query for Stack Instance(s): %v", err)
		}
		maybeForce := ""
		if config.Force {
			maybeForce = "?force=true"
		}
		path := fmt.Sprintf("%s/%s/workerpools/%s%s", stackInstancesResource,
			url.PathEscape(instance.Platform.Id), url.PathEscape(instance.Id), maybeForce)
		code, err := delete(hubApi, path)
		if err != nil {
			log.Fatalf("Error deleting SuperHub `%s` Workerpool `%s`: %v",
				instance.Platform.Domain, instance.Name, err)
		}
		if code != 202 {
			log.Fatalf("Got %d HTTP deleting SuperHub `%s` Workerpool `%s`, expected 202 HTTP",
				code, instance.Platform.Domain, instance.Name)
		}
		if waitAndTailDeployLogs {
			if config.Verbose {
				log.Print("Tailing automation task logs... ^C to interrupt")
			}
			os.Exit(Logs([]string{"stackInstance/" + instance.Id}, true))
		}
	} else {
		_, err := commandStackInstance(selector, "undeploy", nil, waitAndTailDeployLogs, false)
		if err != nil {
			log.Fatalf("Unable to undeploy SuperHub Workerpool: %v", err)
		}
	}
}

func DeleteWorkerpool(selector string) {
	VerifyWorkerpool(selector)
	err := deleteStackInstance(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Workerpool: %v", err)
	}
}
