package state

import (
	"log"
	"time"

	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/storage"
	"hub/util"
)

const statusUpdateDurationThreshold = time.Duration(10 * time.Second)

func UpdateState(manifest *StateManifest,
	componentName, componentStatus, stackStatus string,
	stackParameters parameters.LockedParameters, componentParameters []parameters.LockedParameter,
	rawOutputs parameters.RawOutputs, outputs parameters.CapturedOutputs,
	requestedOutputs []manifest.Output,
	provides map[string][]string,
	final bool) *StateManifest {

	now := time.Now()

	manifest = maybeInitState(manifest)
	componentState := maybeInitComponentState(manifest, componentName)

	if config.Debug {
		if componentStatus != "" {
			log.Printf("State component `%s` status: %s", componentName, componentStatus)
		}
		if stackStatus != "" {
			log.Printf("State stack status: %s", stackStatus)
		}
	}

	componentState.Timestamp = now
	if componentStatus != "" {
		componentState.Status = componentStatus
	}
	componentState.Parameters = componentParameters
	componentState.CapturedOutputs = parameters.CapturedOutputsToList(outputs)
	if len(rawOutputs) > 0 {
		componentState.RawOutputs = parameters.RawOutputsToList(rawOutputs)
	}

	manifest.Timestamp = now
	if stackStatus != "" {
		manifest.Status = stackStatus
	}
	if final {
		manifest.CapturedOutputs = componentState.CapturedOutputs
	}
	manifest.StackParameters = parameters.LockedParametersToList(stackParameters)
	expandedOutputs := parameters.ExpandRequestedOutputs(stackParameters, outputs, requestedOutputs, final)
	manifest.StackOutputs = mergeExpandedOutputs(manifest.StackOutputs, expandedOutputs, requestedOutputs)
	manifest.Provides = provides

	return manifest
}

func UpdateStatus(manifest *StateManifest,
	componentName, componentStatus, componentMessage, stackStatus, message string) (*StateManifest, bool) {

	now := time.Now()
	write := false

	manifest = maybeInitState(manifest)
	if componentName != "" && componentStatus != "" {
		componentState := maybeInitComponentState(manifest, componentName)
		if componentState.Status != componentStatus || componentState.Message != componentMessage || now.After(componentState.Timestamp.Add(statusUpdateDurationThreshold)) {
			componentState.Timestamp = now
			componentState.Status = componentStatus
			componentState.Message = componentMessage
			if config.Debug {
				log.Printf("State component `%s` status: %s", componentName, componentStatus)
				if componentMessage != "" && config.Trace {
					log.Printf("State component `%s` message: %s", componentName, componentMessage)
				}
			}
			write = true
		}
	}
	if stackStatus != "" {
		if manifest.Status != stackStatus || manifest.Message != message || now.After(manifest.Timestamp.Add(statusUpdateDurationThreshold)) {
			manifest.Timestamp = now
			manifest.Status = stackStatus
			manifest.Message = message
			if config.Debug {
				log.Printf("State stack status: %s", stackStatus)
				if message != "" && config.Trace {
					log.Printf("State stack message: %s", message)
				}
			}
			write = true
		}
	}

	return manifest, write
}

func WriteState(manifest *StateManifest, stateFiles *storage.Files) {
	manifest.Version = 1
	manifest.Kind = "state"

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		log.Fatalf("Unable to marshal state into YAML: %v", err)
	}

	errs := storage.Write(yamlBytes, stateFiles)
	if len(errs) > 0 {
		log.Fatalf("Unable to write state: %s", util.Errors2(errs...))
	}
}

func maybeInitState(manifest *StateManifest) *StateManifest {
	if manifest == nil {
		manifest = &StateManifest{}
	}
	if manifest.Components == nil {
		manifest.Components = make(map[string]*StateStep)
	}
	return manifest
}

func maybeInitComponentState(manifest *StateManifest, componentName string) *StateStep {
	componentState, exist := manifest.Components[componentName]
	if !exist {
		componentState = &StateStep{}
		manifest.Components[componentName] = componentState
	}
	return componentState
}

func mergeExpandedOutputs(prev, curr []parameters.ExpandedOutput, requestedOutputs []manifest.Output) []parameters.ExpandedOutput {
	if len(prev) == 0 {
		return curr
	}
	currNames := make([]string, 0, len(curr))
	for _, c := range curr {
		currNames = append(currNames, c.Name)
	}
	reqNames := make([]string, 0, len(requestedOutputs))
	for _, r := range requestedOutputs {
		reqNames = append(reqNames, r.Name)
	}
	for _, p := range prev {
		if !util.Contains(currNames, p.Name) && util.Contains(reqNames, p.Name) {
			curr = append(curr, p)
		}
	}
	return curr
}
