// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package api

import (
	"log"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/kube"
	"github.com/epam/hubctl/cmd/hub/parameters"
	"github.com/epam/hubctl/cmd/hub/state"
	"github.com/epam/hubctl/cmd/hub/storage"
	"github.com/epam/hubctl/cmd/hub/util"
)

func SyncStackInstance(selector, status string, stateFilenames []string) {
	patch := StackInstancePatch{Status: &StackInstanceStatus{Status: status}}

	replace := false
	if len(stateFilenames) > 0 {
		replace = true
		state := state.MustParseStateFiles(stateFilenames)

		patch = TransformStateToApi(state)
		if status != "" {
			patch.Status.Status = status
		}

		remoteStatePaths := storage.RemoteStoragePaths(stateFilenames)
		if len(remoteStatePaths) > 0 {
			patch.StateFiles = remoteStatePaths
		}
	}

	if config.Verbose {
		log.Print("Syncing Stack Instance state to HubCTL")
	}
	patched, err := PatchStackInstance(selector, patch, replace)
	if err != nil {
		log.Fatalf("Unable to sync Stack Instance: %v", err)
	}

	errs := formatStackInstanceEntity(patched, false, false, false, make([]error, 0))
	if len(errs) > 0 {
		config.AggWarnings = false
		util.Warn("%s", util.Errors2(errs...))
	}
}

func TransformStateToApi(state *state.StateManifest) StackInstancePatch {
	return StackInstancePatch{
		ComponentsEnabled: state.Lifecycle.Order,
		Parameters:        transformStackParametersToApi(state.StackParameters),
		Status: &StackInstanceStatus{
			Status:     state.Status,
			Components: transformComponentsToApi(state.Lifecycle.Order, state.Components),
		},
		InflightOperations: transformOperationsToApi(state.Operations),
		Outputs:            transformStackOutputsToApi(appendKubernetesOutputs(state.StackOutputs, state.Components)),
		Provides:           state.Provides,
	}
}

func transformStackParametersToApi(lockedParams []parameters.LockedParameter) []Parameter {
	kv := make(map[string][]parameters.LockedParameter)
	// group parameters by plain name
	for _, p := range lockedParams {
		if strings.HasPrefix(p.Name, "hub.") || util.Empty(p.Value) {
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
	for _, params := range kv {
		// find value of parameter without component: qualifier
		var wildcardValue string
		var wildcardExist bool
		for _, p := range params {
			if p.Component == "" {
				wildcardExist = true
				wildcardValue = util.String(p.Value)
				break
			}
		}
		uniqParams := params
		if wildcardExist {
			// if parameter without component: qualifier exist, then
			// erase all parameters with component: qualifier having same value as wildcard
			uniqParams = uniqParams[:0]
			for _, p := range params {
				if p.Component == "" || util.String(p.Value) != wildcardValue {
					uniqParams = append(uniqParams, p)
				}
			}
		}
		// transform into API representation
		for _, p := range uniqParams {
			var value interface{}
			kind := ""
			if util.LooksLikeSecret(p.Name) || util.Contains(kube.KubernetesSecretParameters, p.Name) {
				secretKind := guessSecretKind("", p.Name)
				value = map[string]string{
					"kind":     secretKind,
					secretKind: util.String(p.Value),
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
		if component, exist := stateComponents[name]; exist && component.Status != "" {
			noSecretOutputs := filterOutSecretOutputs(component.CapturedOutputs)
			outputs := transformComponentOutputsToApi(state.DiffOutputs(noSecretOutputs, prevOutputs))
			if len(noSecretOutputs) > 0 {
				prevOutputs = noSecretOutputs
			}
			version := component.Meta.Version
			if version == "" && component.Version != "" {
				version = component.Version
			}
			components = append(components,
				ComponentStatus{
					Name:    name,
					Status:  component.Status,
					Version: version, // TODO deprecate in favor of Meta.Version
					Meta: &ComponentMetadata{
						Origin:      component.Meta.Origin,
						Kind:        component.Meta.Kind,
						Title:       component.Meta.Title,
						Brief:       component.Meta.Brief,
						Description: component.Meta.Description,
						Version:     component.Meta.Version,
						Maturity:    component.Meta.Maturity,
						Icon:        component.Meta.Icon,
					},
					Message:    component.Message,
					Outputs:    outputs,
					Timestamps: &Timestamps{Start: component.Timestamps.Start, End: component.Timestamps.End},
				})
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

func transformComponentOutputsToApi(outputs []parameters.CapturedOutput) []ComponentOutput {
	apiOutputs := make([]ComponentOutput, 0, len(outputs))
	for _, output := range outputs {
		apiOutputs = append(apiOutputs, ComponentOutput{
			Name:  output.Name,
			Value: output.Value,
			Brief: output.Brief,
		})
	}
	return apiOutputs
}

func appendKubernetesOutputs(outputs []parameters.ExpandedOutput, components map[string]*state.StateStep) []parameters.ExpandedOutput {
	// make sure Kubernetes keys are added to stack outputs if not already there
	for _, providerName := range kube.KubernetesDefaultProviders {
		if provider, exist := components[providerName]; exist {
		next_output:
			for _, outputName := range kube.KubernetesParameters {
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

func transformStackOutputsToApi(stackOutputs []parameters.ExpandedOutput) []Output {
	outputs := make([]Output, 0, len(stackOutputs))
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
		if strings.HasPrefix(o.Kind, "secret") || util.LooksLikeSecret(name) ||
			util.Contains(kube.KubernetesSecretParameters, name) {

			secretKind := guessSecretKind(o.Kind, name)
			value = map[string]string{
				"kind":     secretKind,
				secretKind: util.String(o.Value),
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
	} else if strings.HasSuffix(name, ".token") || strings.HasSuffix(name, "Token") {
		kind = "token"
	} else if strings.HasSuffix(name, ".password") || strings.HasSuffix(name, "Password") {
		kind = "password"
	}
	return kind
}

func transformOperationsToApi(ops []state.LifecycleOperation) []InflightOperation {
	apiOps := make([]InflightOperation, 0, len(ops))
	for _, op := range ops {
		phases := make([]LifecyclePhase, 0, len(op.Phases))
		for _, phase := range op.Phases {
			phases = append(phases, LifecyclePhase{Phase: phase.Phase, Status: phase.Status})
		}
		apiOps = append(apiOps, InflightOperation{
			Id:          op.Id,
			Operation:   op.Operation,
			Timestamp:   op.Timestamp,
			Status:      op.Status,
			Description: op.Description,
			Initiator:   op.Initiator,
			Logs:        op.Logs,
			Phases:      phases,
		})
	}
	return apiOps
}
