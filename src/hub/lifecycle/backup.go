package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/state"
	"hub/storage"
	"hub/util"
)

func BackupCreate(request *Request, bundles []string, jsonOutput, allowPartial bool) {
	if len(request.StateFilenames) == 0 {
		log.Fatal("Backup without state file(s) is not implemented; try --state")
	}

	if len(bundles) == 0 && config.Verbose && !config.Debug {
		config.Verbose = false
		config.AggWarnings = false
	}

	warnBackupFlagsNotImplemented(request)

	stackManifest, componentsManifests, _, err := manifest.ParseManifest(request.ManifestFilenames)
	if err != nil {
		log.Fatalf("Unable to create backup: %v", err)
	}

	osEnv, err := initOsEnv(request.OsEnvironmentMode)
	if err != nil {
		log.Fatalf("Unable to parse OS environment setup: %v", err)
	}

	stackBaseDir := util.Basedir(request.ManifestFilenames)
	componentsBaseDir := request.ComponentsBaseDir
	if componentsBaseDir == "" {
		componentsBaseDir = stackBaseDir
	}

	components := stackManifest.Components
	checkComponentsManifests(components, componentsManifests)
	order := stackManifest.Lifecycle.Order
	if len(request.Components) > 0 {
		manifest.CheckComponentsExist(components, request.Components...)
		components = make([]manifest.ComponentRef, 0, len(request.Components))
		for _, comp := range request.Components {
			componentRef := findComponentRef(stackManifest.Components, comp)
			components = append(components, *componentRef)
		}
		order = request.Components
	}
	checkComponentsSourcesExist(components, stackBaseDir, componentsBaseDir)
	if len(request.Components) == 0 {
		checkLifecycleOrder(components, stackManifest.Lifecycle)
	}

	implementsBackup := make([]string, 0, len(order))
	for _, componentName := range order {
		component := findComponentRef(components, componentName)
		dir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)
		impl := probeImplementation(dir, request.Verb)
		if impl {
			implementsBackup = append(implementsBackup, manifest.ComponentQualifiedNameFromRef(component))
		}
	}
	if len(request.Components) == 0 && len(implementsBackup) == 0 {
		log.Fatalf("No component implements `%s` verb", request.Verb)
	}
	if len(request.Components) > 0 && len(request.Components) != len(implementsBackup) {
		for _, comp := range request.Components {
			if !util.Contains(implementsBackup, comp) {
				log.Printf("Component `%s` does not implement `%s` verb", comp, request.Verb)
			}
		}
		os.Exit(1)
	}

	optionalRequires := parseRequiresTunning(stackManifest.Lifecycle.Requires)
	stackProvides := checkRequires(stackManifest.Requires, optionalRequires)

	stateFiles, errs := storage.Check(request.StateFilenames, "state")
	if len(errs) > 0 {
		maybeFatalf("Unable to check state files: %v", util.Errors2(errs...))
	}

	var bundleFiles *storage.Files
	if len(bundles) > 0 {
		checked, errs := storage.Check(bundles, "backup bundle")
		if len(errs) > 0 {
			maybeFatalf("Unable to check backup bundle files: %v", util.Errors2(errs...))
		}
		bundleFiles = checked
	}

	if config.Verbose {
		printBackupStartBlurb(request, bundles)
	}

	parsedState, err := state.ParseState(stateFiles)
	if err != nil {
		maybeFatalf("Unable to load state %v: %v", request.StateFilenames, err)
	}

	if config.Verbose {
		log.Printf("%s %v", strings.Title(request.Verb), implementsBackup)
	}

	bundle := state.BackupManifest{
		Version:    1,
		Kind:       "backup",
		Components: make(map[string]state.ComponentBackup),
	}
	failedComponents := make([]string, 0)

	for componentIndex, componentName := range implementsBackup {
		if config.Verbose {
			log.Printf("%s ***%s*** (%d/%d)", request.Verb, componentName, componentIndex+1, len(implementsBackup))
		}

		component := findComponentRef(components, componentName)
		componentManifest := findComponentManifest(component, componentsManifests)

		// TODO Should we reload new parameters from elaborate to allow for component's source mismatch?
		// Or it will encourage bad practice?
		stackParameters := make(parameters.LockedParameters)
		allOutputs := make(parameters.CapturedOutputs)
		provides := util.CopyMap2(stackProvides)
		if parsedState != nil {
			state.MergeParsedState(parsedState,
				componentName, component.Depends, stackManifest.Lifecycle.Order, false,
				stackParameters, allOutputs, provides)
		}

		expandedComponentParameters, errs := parameters.ExpandParameters(componentName, component.Depends,
			stackParameters, allOutputs,
			manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name),
			nil)
		if len(errs) > 0 {
			maybeFatalf("Component `%s` parameters expansion failed:\n\t%s",
				componentName, util.Errors("\n\t", errs...))
		}
		componentParameters := make(parameters.LockedParameters)
		if !request.StrictParameters {
			componentParameters = stackParameters
		}
		componentParameters = parameters.MergeParameters(componentParameters, expandedComponentParameters)

		if config.Debug {
			log.Print("Component parameters:")
			parameters.PrintLockedParameters(componentParameters)
		}

		prepareComponentRequires(provides, componentManifest, stackParameters, allOutputs, optionalRequires)

		dir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)
		stdout, err := delegate(request.Verb, component, componentManifest, componentParameters,
			dir, osEnv, request.PipeOutputInRealtime)

		var rawOutputs parameters.RawOutputs = nil
		if len(stdout) > 0 {
			rawOutputs = parseTextOutput(stdout)
		}
		status := "error"
		if err != nil || len(rawOutputs) == 0 {
			if err == nil {
				err = errors.New("no outputs emited by the component")
			}
			log.Printf("Component `%s` failed to %s: %v", componentName, request.Verb, err)
			failedComponents = append(failedComponents, componentName)
			if !allowPartial {
				break
			}
		} else {
			status = "success"
			log.Printf("Component `%s` completed %s", componentName, request.Verb)
		}
		kind, exist := rawOutputs["kind"]
		if !exist || kind == "" {
			kind = componentName
		}
		timestamp := time.Now()
		timestampStr, exist := rawOutputs["timestamp"]
		if exist && timestampStr != "" {
			timestamp2, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				util.Warn("Unable to parse timestamp `%s` emited by component `%s`: %v; using current time",
					timestampStr, componentName, err)
			} else {
				timestamp = timestamp2
			}
		}
		delete(rawOutputs, "kind")
		delete(rawOutputs, "timestamp")
		outputs := make([]parameters.CapturedOutput, 0, len(rawOutputs))
		for name, value := range rawOutputs {
			outputs = append(outputs, parameters.CapturedOutput{Name: name, Value: value})
		}
		bundle.Components[componentName] = state.ComponentBackup{
			Timestamp: timestamp,
			Status:    status,
			Kind:      kind,
			Outputs:   outputs,
		}
	}

	if len(failedComponents) > 0 {
		log.Printf("Component(s) failed to %s: %v", request.Verb, failedComponents)
		if allowPartial && len(failedComponents) < len(implementsBackup) {
			bundle.Status = "partial"
		} else {
			bundle.Status = "error"
		}
	} else {
		bundle.Status = "success"
	}
	bundle.Timestamp = time.Now()

	format := "yaml"
	marshall := yaml.Marshal
	if jsonOutput {
		format = "json"
		marshall = json.Marshal
	}
	bytes, err := marshall(&bundle)
	if err != nil {
		log.Fatalf("Unable to marshal backup bundle into %s: %v", format, err)
	}
	if bundleFiles != nil {
		storage.Write(bytes, bundleFiles)
	} else {
		os.Stdout.Write([]byte(fmt.Sprintf("--- %s\n", format)))
		os.Stdout.Write(bytes)
	}

	if config.Verbose {
		printBackupEndBlurb(request, stackManifest)
	}
}

func warnBackupFlagsNotImplemented(request *Request) {
	if request.Application != "" {
		util.Warn("Application `%s` parameters won't be used - not implemented. Parameters are loaded from state.", request.Application)
	}
	if request.StackInstance != "" {
		util.Warn("Stack Instance `%s` parameters won't be used - not implemented. Parameters are loaded from state.", request.StackInstance)
	}
	if request.Environment != "" {
		util.Warn("Environment `%s` parameters won't be used - not implemented. Parameters are loaded from state.", request.Environment)
	}
}
