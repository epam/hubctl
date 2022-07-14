// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"log"
	"os"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/storage"
	"github.com/agilestacks/hub/cmd/hub/util"
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

	additionalKV, err := util.ParseKvList(additionalParametersStr)
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
		order, err := manifest.GenerateLifecycleOrder(stackManifest)
		if err != nil {
			log.Fatal(err)
		}
		stackManifest.Lifecycle.Order = order
		manifest.CheckComponentsExist(stackManifest.Components, componentName)
		component := manifest.ComponentRefByName(stackManifest.Components, componentName)
		componentManifest := manifest.ComponentManifestByRef(componentsManifests, component)

		stackParameters := make(parameters.LockedParameters)
		outputs = make(parameters.CapturedOutputs)
		_, err = state.MergeState(stateFiles,
			componentName, component.Depends, stackManifest.Lifecycle.Order, false,
			stackParameters, outputs, nil)
		if err != nil {
			util.MaybeFatalf("Failed to read %v state files to load component `%s` state: %v",
				stateFilenames, componentName, err)
		}
		expandedComponentParameters, errs := parameters.ExpandParameters(componentName, componentManifest.Meta.Kind, component.Depends,
			stackParameters, outputs,
			manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name))
		if len(errs) > 0 {
			util.MaybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
				componentName, util.Errors("\n\t", errs...))
		}
		params = parameters.MergeParameters(stackParameters, expandedComponentParameters, additionalParameters)
	} else {
		st, err := state.ParseState(stateFiles)
		if err != nil {
			log.Fatalf("Unable to load state: %s", err)
		}
		stateParameters := st.StackParameters
		stateOutputs := st.CapturedOutputs
		if componentName != "" && st.Components != nil {
			step, exist := st.Components[componentName]
			if exist {
				stateParameters = step.Parameters
				stateOutputs = step.CapturedOutputs
			} else {
				util.Warn("Component `%s` state doesn't exist; using stack-level parameters and outputs instead", componentName)
			}
		}
		params = parameters.ParametersFromList(stateParameters)
		if len(additionalParameters) > 0 {
			params = parameters.MergeParameters(params, additionalParameters)
		}
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
		util.MaybeFatalf("Failed to process `%s` templates:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}
}
