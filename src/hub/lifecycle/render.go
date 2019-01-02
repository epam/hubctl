package lifecycle

import (
	"log"
	"os"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/state"
	"hub/storage"
	"hub/util"
)

func Render(manifestFilenames, stateFilenames []string, componentName,
	templateKind, additionalParametersStr string, templates []string) {

	var stackManifest *manifest.Manifest
	var componentsManifests []manifest.Manifest
	if len(manifestFilenames) > 0 {
		var err error
		stackManifest, componentsManifests, _, err = manifest.ParseManifest(manifestFilenames)
		if err != nil {
			log.Fatalf("Unable to parse: %v", err)
		}
	}

	additionalKV, err := manifest.ParseKvList(additionalParametersStr)
	if err != nil {
		log.Fatalf("Unable to parse additional parameters `%s`: %v", additionalParametersStr, err)
	}

	stateFiles, errs := storage.Check(stateFilenames, "state")
	if len(errs) > 0 {
		log.Fatalf("Unable to check state files: %s", util.Errors2(errs...))
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current working directory: %v", err)
	}

	if config.Debug {
		componentNamePrint := "*stack*"
		if componentName != "" {
			componentNamePrint = componentName
		}
		log.Printf("Render `%s` %s templates %v in `%s` with `%s` additional parameters and %v state",
			componentNamePrint, templateKind, templates, dir, additionalParametersStr, stateFilenames)
	}

	additionalParameters := make([]parameters.LockedParameter, 0, len(additionalKV))
	for k, v := range additionalKV {
		additionalParameters = append(additionalParameters, parameters.LockedParameter{
			Component: componentName,
			Name:      k,
			Value:     v,
		})
	}

	var params parameters.LockedParameters
	var outputs parameters.CapturedOutputs

	if stackManifest != nil && componentName != "" {
		manifest.CheckComponentsExist(stackManifest.Components, componentName)
		component := findComponentRef(stackManifest.Components, componentName)
		componentManifest := findComponentManifest(component, componentsManifests)

		stackParameters := make(parameters.LockedParameters)
		outputs = make(parameters.CapturedOutputs)
		_, err = state.MergeState(stateFiles,
			componentName, component.Depends, stackManifest.Lifecycle.Order, false,
			stackParameters, outputs, nil)
		if err != nil {
			maybeFatalf("Failed to read %v state files to load component `%s` state: %v",
				stateFilenames, componentName, err)
		}
		expandedComponentParameters, errs := parameters.ExpandParameters(componentName, component.Depends,
			stackParameters, outputs,
			manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name),
			nil)
		if len(errs) > 0 {
			maybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
				componentName, util.Errors("\n\t", errs...))
		}
		params = parameters.MergeParameters(stackParameters, expandedComponentParameters, additionalParameters)
	} else {
		state, err := state.ParseState(stateFiles)
		if err != nil {
			log.Fatalf("Unable to load state: %s", err)
		}
		stateParameters := state.StackParameters
		stateOutputs := state.CapturedOutputs
		if componentName != "" && state.Components != nil {
			step, exist := state.Components[componentName]
			if exist {
				stateParameters = step.Parameters
				stateOutputs = step.CapturedOutputs
			} else {
				util.Warn("Component `%s` state doesn't exist; using stack-level parameters and outputs instead", componentName)
			}
		}
		params = parameters.ParametersFromList(stateParameters)
		outputs = parameters.OutputsFromList(stateOutputs)
	}

	if config.Debug {
		log.Print("Render parameters:")
		parameters.PrintLockedParameters(params)
		if len(outputs) > 0 {
			log.Print("---")
			parameters.PrintCapturedOutputs(outputs)
		}
	}

	templateSetup := manifest.TemplateSetup{
		Kind:  templateKind,
		Files: templates,
	}

	if componentName == "" {
		componentName = "*stack*"
	}
	ref := &manifest.ComponentRef{Name: componentName}
	errs = processTemplates(ref, &templateSetup, params, outputs, dir)
	if len(errs) > 0 {
		maybeFatalf("Failed to process `%s` templates:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}
}
