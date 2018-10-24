package state

import (
	"log"
	"time"

	"gopkg.in/yaml.v2"

	"hub/manifest"
	"hub/parameters"
	"hub/storage"
	"hub/util"
)

func WriteState(manifest *StateManifest, stateFiles *storage.Files, componentName string,
	stackParameters parameters.LockedParameters, componentParameters []parameters.LockedParameter,
	rawOutputs parameters.RawOutputs, outputs parameters.CapturedOutputs, requestedOutputs []manifest.Output,
	provides map[string][]string,
	final bool, compressed bool) *StateManifest {

	if manifest == nil {
		manifest = &StateManifest{}
	}

	outputsList := parameters.CapturedOutputsToList(outputs)
	expandedOutputs := parameters.ExpandRequestedOutputs(stackParameters, outputs, requestedOutputs, final)

	if manifest.Components == nil {
		manifest.Components = make(map[string]StateStep)
	}

	rawOutputsList := parameters.RawOutputsToList(rawOutputs)
	if rawOutputs == nil {
		if prevDeploy, exist := manifest.Components[componentName]; exist {
			rawOutputsList = prevDeploy.RawOutputs
		}
	}

	manifest.Components[componentName] = StateStep{
		Timestamp:       time.Now(),
		Parameters:      componentParameters,
		RawOutputs:      rawOutputsList,
		CapturedOutputs: outputsList,
	}
	manifest.StackParameters = parameters.LockedParametersToList(stackParameters)
	if final {
		manifest.CapturedOutputs = outputsList
	}
	manifest.StackOutputs = mergeExpandedOutputs(manifest.StackOutputs, expandedOutputs, requestedOutputs)
	manifest.Provides = provides

	manifest.Version = 1
	manifest.Kind = "state"
	manifest.Timestamp = time.Now()

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		log.Fatalf("Unable to marshal state into YAML: %v", err)
	}

	if compressed {
		yamlBytes, err = util.Gzip(yamlBytes)
		if err != nil {
			log.Fatalf("Unable to compress state file data: %v", err)
		}
	}

	errs := storage.Write(yamlBytes, stateFiles)
	if len(errs) > 0 {
		log.Fatalf("Unable to write state: %s", util.Errors2(errs...))
	}

	return manifest
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
