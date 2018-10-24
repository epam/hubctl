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

func Render(manifestFilenames []string, stateFilenames []string, componentName,
	templateKind, additionalParametersStr string, templates []string) {

	stackManifest, componentsManifests, _, err := manifest.ParseManifest(manifestFilenames)
	if err != nil {
		log.Fatalf("Unable to parse: %v", err)
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
		log.Printf("Render `%s` %s templates %v in `%s` with `%s` additional parameters and %v state",
			componentName, templateKind, templates, dir, additionalParametersStr, stateFiles)
	}

	manifest.CheckComponentsExist(stackManifest.Components, componentName)
	component := findComponentRef(stackManifest.Components, componentName)
	componentManifest := findComponentManifest(component, componentsManifests)

	stackParameters := make(parameters.LockedParameters)
	outputs := make(parameters.CapturedOutputs)
	_, err = state.MergeState(stateFiles,
		componentName, component.Depends, stackManifest.Lifecycle.Order, false,
		stackParameters, outputs, nil)
	if err != nil {
		maybeFatalf("Failed to read %v state files to load component `%s` state: %v",
			stateFilenames, componentName, err)
	}
	// we should probably ask mergeState() to load true component parameters
	// from state instead of re-evaluating them here
	expandedComponentParameters, errs := parameters.ExpandParameters(componentName, component.Depends,
		stackParameters, outputs,
		manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name),
		nil)
	if len(errs) > 0 {
		maybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}

	additionalParameters := make([]parameters.LockedParameter, 0, len(additionalKV))
	for k, v := range additionalKV {
		additionalParameters = append(additionalParameters, parameters.LockedParameter{
			Component: componentName,
			Name:      k,
			Value:     v,
		})
	}

	componentParameters := parameters.MergeParameters(stackParameters, expandedComponentParameters, additionalParameters)

	if config.Debug {
		log.Print("Render parameters:")
		parameters.PrintLockedParameters(componentParameters)
	}

	templateSetup := manifest.TemplateSetup{
		Kind:  templateKind,
		Files: templates,
	}

	errs = processTemplates(component, &templateSetup, componentParameters, outputs, dir)
	if len(errs) > 0 {
		maybeFatalf("Failed to process component `%s` templates:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}

}
