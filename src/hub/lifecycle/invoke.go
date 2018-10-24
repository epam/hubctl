package lifecycle

import (
	"fmt"
	"log"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/state"
	"hub/storage"
	"hub/util"
)

func Invoke(request *Request) {
	stackManifest, componentsManifests, _, err := manifest.ParseManifest(request.ManifestFilenames)
	if err != nil {
		log.Fatalf("Unable to parse: %v", err)
	}

	additionalEnvironment, err := manifest.ParseKvList(request.EnvironmentOverrides)
	if err != nil {
		log.Fatalf("Unable to parse additional environment variables `%s`: %v", request.EnvironmentOverrides, err)
	}

	osEnv, err := initOsEnv(request.OsEnvironmentMode)
	if err != nil {
		log.Fatalf("Unable to parse OS environment setup: %v", err)
	}

	stateFiles, errs := storage.Check(request.StateFilenames, "state")
	if len(errs) > 0 {
		maybeFatalf("Unable to check state file: %v", util.Errors2(errs...))
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
	component := findComponentRef(stackManifest.Components, request.Component)
	componentName := manifest.ComponentQualifiedNameFromRef(component)
	componentManifest := findComponentManifest(component, componentsManifests)
	checkVerbs(request.Component, componentManifest.Lifecycle.Verbs, request.Verb)

	stackParameters := make(parameters.LockedParameters)
	outputs := make(parameters.CapturedOutputs)
	_, err = state.MergeState(stateFiles,
		request.Component, component.Depends, stackManifest.Lifecycle.Order, false,
		stackParameters, outputs, nil)
	if err != nil {
		maybeFatalf("Unable to load component `%s` state: %v",
			request.Component, err)
	}
	// we should probably ask mergeState() to load true component parameters
	// from state instead of re-evaluating them here
	expandedComponentParameters, errs := parameters.ExpandParameters(componentName, component.Depends,
		stackParameters, outputs,
		manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name),
		additionalEnvironment)
	if len(errs) > 0 {
		maybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
			componentName, util.Errors("\n\t", errs...))
	}
	// always --strict-parameters
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
	stdout, stderr, err := execImplementation(impl, request.PipeOutputInRealtime)

	if !request.PipeOutputInRealtime && (config.Trace || err != nil) {
		stdoutMsg := formatStdout("stdout", stdout)
		stderrMsg := formatStdout("stderr", stderr)
		if err != nil {
			maybeFatalf("%s%sFailed to %s %s: %v", stdoutMsg, stderrMsg, request.Verb, request.Component, err)
		} else {
			log.Printf("%s%s", stdoutMsg, stderrMsg)
		}
	}
	if request.PipeOutputInRealtime && err != nil {
		maybeFatalf("Failed to %s %s: %v", request.Verb, request.Component, err)
	}
}

func additionalEnvironmentToList(env map[string]string) []string {
	list := make([]string, 0, len(env))
	for k, v := range env {
		list = append(list, fmt.Sprintf("%s=%s", k, v))
	}
	return list
}
