package api

import (
	"log"
	"strings"

	"hub/config"
	"hub/kube"
	"hub/parameters"
	"hub/state"
	"hub/util"
)

func SyncStackInstance(selector string, stateFilenames []string) {
	st := state.MustParseStateFiles(stateFilenames)

	patch := StackInstancePatch{
		Status:   StackInstanceStatus{Status: "deployed"},
		Outputs:  TransformStackOutputsToApi(st.StackOutputs),
		Provides: st.Provides,
	}
	if config.Verbose {
		log.Print("Syncing instance state to Control Plane")
	}
	patched, err := PatchStackInstance(selector, patch)
	if err != nil {
		log.Fatalf("Unable to sync Stack Instance: %v", err)
	}
	errs := formatStackInstanceEntity(patched, false, false, make([]error, 0))
	if len(errs) > 0 {
		config.AggWarnings = false
		util.Warn("%v", err)
	}
}

func TransformStackOutputsToApi(stackOutputs []parameters.ExpandedOutput) []Output {
	outputs := make([]Output, 0, len(stackOutputs))
	kubeSecretOutputs := kube.RequiredKubernetesKeysParameters()
	for _, o := range stackOutputs {
		name := o.Name
		component := ""
		if strings.Contains(o.Name, ":") {
			parts := strings.SplitN(o.Name, ":", 2)
			if len(parts) > 1 {
				component = parts[0]
				name = parts[1]
			}
		}

		var value interface{}
		kind := ""
		if strings.HasPrefix(o.Kind, "secret") || util.Contains(kubeSecretOutputs, name) {
			secretKind := guessOutputSecretKind(o.Kind, name)
			value = map[string]string{
				"kind":     secretKind,
				secretKind: o.Value,
			}
			kind = "secret"
		} else {
			value = o.Value
		}

		outputs = append(outputs, Output{
			Name:      name,
			Component: component,
			Value:     value,
			Kind:      kind,
			Brief:     o.Brief,
			Messenger: "cli",
		})
	}
	return outputs
}

func guessOutputSecretKind(outputKind, name string) string {
	if strings.Contains(outputKind, "/") {
		parts := strings.SplitN(outputKind, "/", 2)
		if len(parts) > 1 && parts[1] != "" {
			return parts[1]
		}
	}

	kind := "text"
	if strings.HasSuffix(name, ".key") || strings.HasSuffix(name, "Key") {
		kind = "privateKey"
	} else if strings.HasSuffix(name, ".cert") || strings.HasSuffix(name, "Cert") {
		kind = "certificate"
	} else if strings.HasSuffix(name, ".password") || strings.HasSuffix(name, "Password") {
		kind = "password"
	}
	return kind
}
