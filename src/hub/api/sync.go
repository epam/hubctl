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

func SyncStackInstance(selector, status string, stateFilenames []string) {
	var st *state.StateManifest
	var s3StatePaths []string
	var componentsEnabled []string
	if len(stateFilenames) > 0 {
		st = state.MustParseStateFiles(stateFilenames)

		componentsEnabled = st.Lifecycle.Order

		for _, filename := range stateFilenames {
			if strings.HasPrefix(filename, "s3://") {
				s3StatePaths = append(s3StatePaths, filename)
			}
		}
	}

	var params []Parameter
	var outputs []Output
	var components []ComponentStatus
	var provides map[string][]string
	if st != nil {
		if status == "" {
			status = st.Status
		}
		params = transformStackParametersToApi(st.StackParameters)
		outputs = TransformStackOutputsToApi(appendKubernetesKeys(st.StackOutputs, st.Components))
		components = transformComponentsToApi(st.Lifecycle.Order, st.Components)
		provides = st.Provides
	}

	patch := StackInstancePatch{
		Parameters:        params,
		ComponentsEnabled: componentsEnabled,
		StateFiles:        s3StatePaths,
		Status:            &StackInstanceStatus{Status: status, Components: components},
		Outputs:           outputs,
		Provides:          provides,
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
		util.Warn("%s", util.Errors2(errs...))
	}
}

func transformStackParametersToApi(lockedParams []parameters.LockedParameter) []Parameter {
	kv := make(map[string][]parameters.LockedParameter)
	// group parameters by plain name
	for _, p := range lockedParams {
		if strings.HasPrefix(p.Name, "hub.") || p.Value == "" {
			// TODO how to preserve user supplied hub.deploymentId?
			// what about empty: allow?
			continue
		}
		if list, exist := kv[p.Name]; exist {
			kv[p.Name] = append(list, p)
		} else {
			kv[p.Name] = []parameters.LockedParameter{p}
		}
	}
	out := make([]Parameter, 0, len(lockedParams))
	kubeSecretOutputs := kube.RequiredKubernetesKeysParameters()
	for _, params := range kv {
		// find value of parameter without component: qualifier
		var wildcardValue string
		var wildcardExist bool
		for _, p := range params {
			if p.Component == "" {
				wildcardExist = true
				wildcardValue = p.Value
				break
			}
		}
		uniqParams := params
		if wildcardExist {
			// if parameter without component: qualifier exist, then
			// erase all parameters with component: qualifier having same value as wildcard
			uniqParams = uniqParams[:0]
			for _, p := range params {
				if p.Component == "" || p.Value != wildcardValue {
					uniqParams = append(uniqParams, p)
				}
			}
		}
		// transform into API representation
		for _, p := range uniqParams {
			var value interface{}
			kind := ""
			if util.LooksLikeSecret(p.Name) || util.Contains(kubeSecretOutputs, p.Name) {
				secretKind := guessSecretKind("", p.Name)
				value = map[string]string{
					"kind":     secretKind,
					secretKind: p.Value,
				}
				kind = "secret"
			} else {
				value = p.Value
			}

			out = append(out, Parameter{
				Name:      p.Name,
				Component: p.Component,
				Kind:      kind,
				Value:     value,
				Messenger: "cli",
			})
		}
	}
	return out
}

func transformComponentsToApi(order []string, stateComponents map[string]*state.StateStep) []ComponentStatus {
	components := make([]ComponentStatus, 0, len(stateComponents))
	var prevOutputs []parameters.CapturedOutput
	for _, name := range order {
		if component, exist := stateComponents[name]; exist {
			noSecretOutputs := filterOutSecretOutputs(component.CapturedOutputs)
			outputs := state.DiffOutputs(noSecretOutputs, prevOutputs)
			prevOutputs = noSecretOutputs
			components = append(components, ComponentStatus{Name: name, Status: component.Status, Outputs: outputs})
		}
	}
	return components
}

func filterOutSecretOutputs(outputs []parameters.CapturedOutput) []parameters.CapturedOutput {
	secrets := 0
	for _, o := range outputs {
		if strings.HasPrefix(o.Kind, "secret") {
			secrets++
		}
	}
	if secrets == 0 {
		return outputs
	}
	if len(outputs) == secrets {
		return []parameters.CapturedOutput{}
	}
	filtered := make([]parameters.CapturedOutput, 0, len(outputs)-secrets)
	for _, o := range outputs {
		if !strings.HasPrefix(o.Kind, "secret") {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

func appendKubernetesKeys(outputs []parameters.ExpandedOutput, components map[string]*state.StateStep) []parameters.ExpandedOutput {
	// make sure Kubernetes keys are added to stack outputs if not already there
	for _, providerName := range kube.KubernetesDefaultProviders {
		if provider, exist := components[providerName]; exist {
		next_output:
			for _, outputName := range kube.RequiredKubernetesKeysParameters() {
				outputQName := parameters.OutputQualifiedName(outputName, providerName)
				for _, stackOutput := range outputs {
					if stackOutput.Name == outputName ||
						stackOutput.Name == outputQName {
						continue next_output // already present on stack outputs
					}
				}
				for _, output := range provider.CapturedOutputs {
					if output.Name == outputName {
						outputs = append(outputs, parameters.ExpandedOutput{Name: outputQName, Value: output.Value})
					}
				}
			}
			break
		}
	}
	return outputs
}

func TransformStackOutputsToApi(stackOutputs []parameters.ExpandedOutput) []Output {
	outputs := make([]Output, 0, len(stackOutputs))
	kubeSecretOutputs := kube.RequiredKubernetesKeysParameters()
	for _, o := range stackOutputs {
		name := o.Name
		component := ""
		if strings.Contains(name, ":") {
			parts := strings.SplitN(name, ":", 2)
			if len(parts) > 1 {
				component = parts[0]
				name = parts[1]
			}
		}

		var value interface{}
		kind := ""
		if strings.HasPrefix(o.Kind, "secret") || util.LooksLikeSecret(name) || util.Contains(kubeSecretOutputs, name) {
			secretKind := guessSecretKind(o.Kind, name)
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

func guessSecretKind(outputKind, name string) string {
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
