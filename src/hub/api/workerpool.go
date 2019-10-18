package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	workerpoolSpotPrice float32, workerpoolPreemptibleVMs, workerpoolAutoscale bool, workerpoolVolumeSize int,
	waitAndTailDeployLogs, dryRun bool) {

	log.Fatal("Not implemented")
}

func ScaleWorkerpool(selector string, count, maxCount int, waitAndTailDeployLogs, dryRun bool) {
	log.Fatal("Not implemented")
}

func DeployWorkerpool(selector string, waitAndTailDeployLogs, dryRun bool) {
	log.Fatal("Not implemented")
}

func UndeployWorkerpool(selector string, waitAndTailDeployLogs bool) {
	log.Fatal("Not implemented")
}

func DeleteWorkerpool(selector string) {
	log.Fatal("Not implemented")
}
