package lifecycle

import (
	"fmt"
	"log"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/storage"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func Invoke(request *Request) {
	stackManifest, componentsManifests, _, err := manifest.ParseManifest(request.ManifestFilenames)
	if err != nil {
		log.Fatalf("Unable to parse: %v", err)
	}

	additionalEnvironment, err := util.ParseKvList(request.EnvironmentOverrides)
	if err != nil {
		log.Fatalf("Unable to parse additional environment variables `%s`: %v", request.EnvironmentOverrides, err)
	}

	osEnv, err := initOsEnv(request.OsEnvironmentMode)
	if err != nil {
		log.Fatalf("Unable to parse OS environment setup: %v", err)
	}

	stateFiles, errs := storage.Check(request.StateFilenames, "state")
	if len(errs) > 0 {
		util.MaybeFatalf("Unable to check state file: %v", util.Errors2(errs...))
	}

	if config.Verbose {
		log.Printf("Invoke `%s` on `%s` with %v manifest and %v state",
			request.Verb, request.Component, request.ManifestFilenames, request.StateFilenames)
	}

	stackBaseDir := util.Basedir(request.ManifestFilenames)
	componentsBaseDir := request.ComponentsBaseDir
	if componentsBaseDir == "" {
		componentsBaseDir = stackBaseDir
	}

	manifest.CheckComponentsExist(stackManifest.Components, request.Component)
	component := manifest.ComponentRefByName(stackManifest.Components, request.Component)
	componentName := manifest.ComponentQualifiedNameFromRef(component)
	componentManifest := manifest.ComponentManifestByRef(componentsManifests, component)
	checkVerbs(request.Component, componentManifest.Lifecycle.Verbs, request.Verb)

	stackParameters := make(parameters.LockedParameters)
	outputs := make(parameters.CapturedOutputs)
	_, err = state.MergeState(stateFiles,
		request.Component, component.Depends, stackManifest.Lifecycle.Order, false,
		stackParameters, outputs, nil)
	if err != nil {
		util.MaybeFatalf("Unable to load component `%s` state: %v",
			request.Component, err)
	}
	// should we ask mergeState() to load true component parameters
	// from state instead of re-evaluating them here?
	expandedComponentParameters, errs := parameters.ExpandParameters(componentName, componentManifest.Meta.Kind, component.Depends,
		stackParameters, outputs, manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name))
	if len(errs) > 0 {
		util.MaybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}
	componentParameters := parameters.MergeParameters(make(parameters.LockedParameters), expandedComponentParameters)

	if config.Debug {
		log.Print("Component parameters:")
		parameters.PrintLockedParameters(componentParameters)
	}

	dir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)
	if config.Debug {
		log.Printf("Component `%s` directory: %s", request.Component, dir)
	}
	impl, err := findImplementation(dir, request.Verb)
	if err != nil {
		log.Fatalf("Failed to %s %s: %v", request.Verb, request.Component, err)
	}
	processEnv := mergeOsEnviron(
		parametersInEnv(componentName, componentParameters),
		additionalEnvironmentToList(additionalEnvironment))
	impl.Env = mergeOsEnviron(osEnv, processEnv)
	if config.Debug && len(processEnv) > 0 {
		log.Print("Component environment:")
		printEnvironment(processEnv)
		if config.Trace {
			log.Print("Full process environment:")
			printEnvironment(impl.Env)
		}
	}

	_, _, err = execImplementation(impl, true, false)

	if err != nil {
		util.MaybeFatalf("Failed to %s %s: %v", request.Verb, request.Component, err)
	}
}

func additionalEnvironmentToList(env map[string]string) []string {
	list := make([]string, 0, len(env))
	for k, v := range env {
		list = append(list, fmt.Sprintf("%s=%s", k, v))
	}
	return list
}
