// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/ext"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/epam/hubctl/cmd/hub/parameters"
	"github.com/epam/hubctl/cmd/hub/state"
	"github.com/epam/hubctl/cmd/hub/storage"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	HubEnvVarNameComponentName    = "HUB_COMPONENT"
	HubEnvVarNameComponentDir     = "HUB_COMPONENT_DIR"
	HubEnvVarNameStackBasedir     = "HUB_BASE_DIR"
	HubEnvVarNameRandom           = "HUB_RANDOM"
	SkaffoldKubeContextEnvVarName = "SKAFFOLD_KUBE_CONTEXT"
	HubEnvVarHubStackName         = "HUB_STACK_NAME"
)

func Execute(request *Request, pipe io.WriteCloser) {
	isDeploy := strings.HasPrefix(request.Verb, "deploy")
	isUndeploy := strings.HasPrefix(request.Verb, "undeploy")
	isSomeComponents := len(request.Components) > 0 || request.OffsetComponent != ""

	stackManifest, componentsManifests, chosenManifestFilename, err := manifest.ParseManifest(request.ManifestFilenames)
	if err != nil {
		log.Fatalf("Unable to %s: %s", request.Verb, err)
	}

	if pipe != nil {
		metricTags := fmt.Sprintf("stack:%s", stackManifest.Meta.Name)
		pipe.Write([]byte(metricTags))
		pipe.Close()
	}

	environment, err := util.ParseKvList(request.EnvironmentOverrides)
	if err != nil {
		log.Fatalf("Unable to parse environment settings `%s`: %v", request.EnvironmentOverrides, err)
	}

	order, err := manifest.GenerateLifecycleOrder(stackManifest)
	if err != nil {
		log.Fatal(err)
	}
	stackManifest.Lifecycle.Order = order

	if config.Verbose {
		printStartBlurb(request, chosenManifestFilename, stackManifest)
	}

	stackBaseDir := util.Basedir(request.ManifestFilenames)
	componentsBaseDir := request.ComponentsBaseDir
	if componentsBaseDir == "" {
		componentsBaseDir = stackBaseDir
	}

	components := stackManifest.Components
	checkComponentsManifests(components, componentsManifests)
	checkLifecycleOrder(components, stackManifest.Lifecycle)
	checkLifecycleRequires(components, stackManifest.Lifecycle.Requires)
	checkComponentsDepends(components, stackManifest.Lifecycle.Order)
	manifest.CheckComponentsExist(components, append(request.Components, request.OffsetComponent, request.LimitComponent)...)
	optionalRequires := parseRequiresTunning(stackManifest.Lifecycle.Requires)
	requiresOfOptionalComponents := calculateRequiresOfOptionalComponents(componentsManifests, &stackManifest.Lifecycle, stackManifest.Requires)
	stackRequires := maybeOmitCloudRequires(stackManifest.Requires, request.EnabledClouds)
	provides := checkStackRequires(stackRequires, optionalRequires, requiresOfOptionalComponents)
	mergePlatformProvides(provides, stackManifest.Platform.Provides)
	if config.Debug && len(provides) > 0 {
		log.Print("Requirements provided by:")
		util.PrintDeps(provides)
	}

	osEnv, err := initOsEnv(request.OsEnvironmentMode)
	if err != nil {
		log.Fatalf("Unable to parse OS environment setup: %v", err)
	}

	defer util.Done()

	var stateManifest *state.StateManifest
	var operationsHistory []state.LifecycleOperation
	stateUpdater := func(interface{}) {}
	var operationLogId string
	if len(request.StateFilenames) > 0 {
		stateFiles, errs := storage.Check(request.StateFilenames, "state")
		if len(errs) > 0 {
			util.MaybeFatalf("Unable to check state files: %s", util.Errors2(errs...))
		}
		storage.EnsureNoLockFiles(stateFiles)
		parsed, err := state.ParseState(stateFiles)
		if isUndeploy || isSomeComponents {
			if err != nil {
				if err != os.ErrNotExist {
					log.Fatalf("Failed to read %v state files: %v", request.StateFilenames, err)
				}
				if isSomeComponents {
					comps := request.OffsetComponent
					if comps == "" {
						comps = strings.Join(request.Components, ", ")
					}
					util.MaybeFatalf("Component `%s` is specified but failed to read %v state file(s): %v",
						comps, request.StateFilenames, err)
				}
			} else {
				stateManifest = parsed
			}
		} else { // full deploy copy state oplog if possible
			if err == nil && parsed != nil {
				if config.Debug {
					log.Print("Preserving operations history loaded from existing state")
				}
				operationsHistory = parsed.Operations
				if parsed.Meta.Name != "" && parsed.Meta.Name != stackManifest.Meta.Name {
					util.Warn("State meta.name = `%s` does not match elaborate meta.name = `%s`",
						parsed.Meta.Name, stackManifest.Meta.Name)
				}
				stateManifest = parsed
			}
		}
		var syncer func(*state.StateManifest)
		// TODO sync status if no state manifest on undeploy
		if request.SyncStackInstance && request.StackInstance != "" {
			syncer = hubSyncer(request)
		}
		stateUpdater = state.InitWriter(stateFiles, syncer)

		u, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Unable to generate operation Id random v4 UUID: %v", err)
		}
		operationLogId = u.String()
	}

	deploymentIdParameterName := "hub.deploymentId"
	deploymentId := ""
	if stateManifest != nil {
		for _, p := range stateManifest.StackParameters {
			if p.Name == deploymentIdParameterName {
				deploymentId = util.String(p.Value)
				break
			}
		}
	}
	if deploymentId == "" {
		u, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Unable to generate `hub.deploymentId` random v4 UUID: %v", err)
		}
		deploymentId = u.String()
	}

	stackNameParameterName := "hub.stackName"
	stackName := ""
	if stateManifest != nil {
		for _, p := range stateManifest.StackParameters {
			if p.Name == stackNameParameterName {
				stackName = util.String(p.Value)
				break
			}
		}
	}
	if stackName == "" {
		stackName = os.Getenv(HubEnvVarHubStackName)
		if stackName == "" {
			rand.Seed(time.Now().UnixNano())
			suffix := rand.Intn(1000) + 1
			name := petname.Generate(2, "-")
			stackName = fmt.Sprintf("%s-%d", name, suffix)
		}
	}

	extraExpansionValues := []manifest.Parameter{
		{Name: deploymentIdParameterName, Value: deploymentId},
		{Name: stackNameParameterName, Value: stackName},
	}

	// TODO state file has user-level parameters for undeploy operation
	// should we just go with the state values if we cannot lock all parameters properly?
	stackParameters, errs := parameters.LockParameters(
		manifest.FlattenParameters(stackManifest.Parameters, chosenManifestFilename),
		extraExpansionValues,
		func(parameter manifest.Parameter) (interface{}, error) {
			return AskParameter(parameter, environment,
				request.Environment, request.StackInstance, request.Application,
				isDeploy)
		})
	if len(errs) > 0 {
		log.Fatalf("Failed to lock stack parameters:\n\t%s", util.Errors("\n\t", errs...))
	}
	allOutputs := make(parameters.CapturedOutputs)
	if stateManifest != nil {
		checkStateMatch(stateManifest, stackManifest, stackParameters)
		state.MergeParsedStateParametersAndProvides(stateManifest, stackParameters, provides)
		if isUndeploy && !isSomeComponents {
			state.MergeParsedStateOutputs(stateManifest,
				"", []string{}, stackManifest.Lifecycle.Order, false,
				allOutputs)
		}
	}
	if stateManifest == nil && isDeploy {
		stateManifest = &state.StateManifest{
			Meta: state.Metadata{
				Kind: stackManifest.Kind,
				Name: stackManifest.Meta.Name,
			},
			Operations: operationsHistory,
		}
	}
	addLockedParameter(stackParameters, deploymentIdParameterName, "DEPLOYMENT_ID", deploymentId)
	addLockedParameter(stackParameters, stackNameParameterName, "STACK_NAME", stackName)
	stackParametersNoLinks := parameters.ParametersWithoutLinks(stackParameters)

	order = stackManifest.Lifecycle.Order
	if stateManifest != nil {
		stateManifest.Lifecycle.Order = order
	}

	offsetGuessed := false
	if isUndeploy {
		order = util.Reverse(order)

		// on undeploy, guess which component failed to deploy and start undeploy from it
		if request.GuessComponent && stateManifest != nil && !isSomeComponents {
			for i, component := range order {
				if _, exist := stateManifest.Components[component]; exist {
					if i == 0 {
						break
					}
					if config.Verbose {
						log.Printf("State file has a state for `%[1]s` - setting `--offset %[1]s`", component)
					}
					request.OffsetComponent = component
					offsetGuessed = true
					break
				}
			}
		}
	}

	offsetComponentIndex := util.Index(order, request.OffsetComponent)
	limitComponentIndex := util.Index(order, request.LimitComponent)
	if offsetComponentIndex >= 0 && limitComponentIndex >= 0 &&
		limitComponentIndex < offsetComponentIndex && !offsetGuessed {
		log.Fatalf("Specified --limit %s (%d) is before specified --offset %s (%d) in component order",
			request.LimitComponent, limitComponentIndex, request.OffsetComponent, offsetComponentIndex)
	}
	skipComponent := func(i int, name string) bool {
		return (len(request.Components) > 0 && !util.Contains(request.Components, name)) ||
			(offsetComponentIndex >= 0 && i < offsetComponentIndex) ||
			(limitComponentIndex >= 0 && i > limitComponentIndex)
	}
	checkComponentsSourcesExist(order, components, stackBaseDir, componentsBaseDir, skipComponent)
	checkLifecycleVerbs(order, components, componentsManifests, stackManifest.Lifecycle.Verbs, stackBaseDir, componentsBaseDir, skipComponent)

	failedComponents := make([]string, 0)

	// TODO handle ^C interrupt to update op log and stack status
	// or expiry by time and set to `interrupted`
	if stateManifest != nil {
		stateManifest = state.UpdateOperation(stateManifest, operationLogId, request.Verb, "in-progress",
			map[string]interface{}{"args": os.Args})
	}

	ctx := watchInterrupt()

NEXT_COMPONENT:
	for componentIndex, componentName := range order {
		if skipComponent(componentIndex, componentName) {
			if config.Debug {
				log.Printf("Skip %s", componentName)
			}
			continue
		}

		if config.Verbose {
			log.Printf(util.HighlightColor("%s ***%s*** (%d/%d)"), maybeTestVerb(request.Verb, request.DryRun),
				componentName, componentIndex+1, len(components))
		}

		component := manifest.ComponentRefByName(components, componentName)
		componentManifest := manifest.ComponentManifestByRef(componentsManifests, component)

		if stateManifest != nil && (componentIndex == offsetComponentIndex || len(request.Components) > 0) {
			if len(request.Components) > 0 {
				allOutputs = make(parameters.CapturedOutputs)
			}
			state.MergeParsedStateOutputs(stateManifest,
				componentName, component.Depends, stackManifest.Lifecycle.Order, isDeploy,
				allOutputs)
		}

		var updateStateComponentFailed func(string, bool)
		if stateManifest != nil {
			stateManifest = state.UpdateComponentStartTimestamp(stateManifest, componentName)
			updateStateComponentFailed = func(msg string, final bool) {
				stateManifest = state.UpdateComponentStatus(stateManifest, componentName, &componentManifest.Meta, "error", msg)
				stateManifest = state.UpdatePhase(stateManifest, operationLogId, componentName, "error")
				// Erasing provides of a failed component on redeploy has undesirable effect on undeploy, for example:
				// Kubernetes Terraform failed safely - failed to download plugin, or failed to add minor resource - leaving
				// Kubernetes fully operation, yet the `kubernetes` capability is removed from stack provides. Such stack
				// will fail to undeploy due to components being unable to satisfy the [kubernetes] requirement, ie.
				// requiring deploy to undeploy, which is dumb.
				// eraseProvides(provides, componentName)
				stateManifest.Provides = noEnvironmentProvides(provides)
				if !config.Force && !optionalComponent(&stackManifest.Lifecycle, componentName) {
					stateManifest = state.UpdateStackStatus(stateManifest, "incomplete", msg)
				}
				if final {
					stateManifest = state.UpdateOperation(stateManifest, operationLogId, request.Verb, "error", nil)
				}
				stateUpdater(stateManifest)
			}
		}

		if isDeploy && len(component.Depends) > 0 {
			failed := make([]string, 0, len(component.Depends))
			for _, dependency := range component.Depends {
				if util.Contains(failedComponents, dependency) {
					failed = append(failed, dependency)
				}
			}
			if len(failed) > 0 {
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
					fmt.Sprintf("Component `%s` failed to %s: depends on failed optional component `%s`",
						componentName, request.Verb, strings.Join(failed, ", ")),
					updateStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
				continue NEXT_COMPONENT
			}
		}

		expandedComponentParameters, expansionErrs := parameters.ExpandParameters(componentName, componentManifest.Meta.Kind, component.Depends,
			stackParameters, allOutputs,
			manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name))
		expandedComponentParameters = addHubProvides(expandedComponentParameters, provides)
		allParameters := parameters.MergeParameters(stackParametersNoLinks, expandedComponentParameters)
		optionalParametersFalse := calculateOptionalFalseParameters(componentName, allParameters, optionalRequires)
		if len(optionalParametersFalse) > 0 {
			log.Printf("Surprisingly, let skip `%s` due to optional parameter %v evaluated to false", componentName, optionalParametersFalse)
			if stateManifest != nil {
				stateManifest = state.EraseComponentEmptyState(stateManifest, componentName)
			}
			continue NEXT_COMPONENT
		}
		if len(expansionErrs) > 0 {
			log.Printf("Component `%s` failed to %s", componentName, request.Verb)
			maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
				fmt.Sprintf("Component `%s` parameters expansion failed:\n\t%s",
					componentName, util.Errors("\n\t", expansionErrs...)),
				updateStateComponentFailed)
			failedComponents = append(failedComponents, componentName)
			continue NEXT_COMPONENT
		}

		componentParameters := parameters.MergeParameters(make(parameters.LockedParameters), expandedComponentParameters)

		if optionalNotProvided, err := prepareComponentRequires(provides, componentManifest, allParameters, allOutputs, optionalRequires, request.EnabledClouds); len(optionalNotProvided) > 0 || err != nil {
			if err != nil {
				if request.Verb == "undeploy" {
					// proceed without --force set to handle required component (depends on) being already undeployed via --component
					util.Warn("%v", err)
				} else {
					maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName, fmt.Sprintf("%v", err), updateStateComponentFailed)
					continue NEXT_COMPONENT
				}
			}
			if len(optionalNotProvided) > 0 {
				log.Printf("Skip %s due to unsatisfied optional requirements %v", componentName, optionalNotProvided)
				// there will be a gap in state file but `deploy -c` will be able to find some state from
				// a preceding component
				continue NEXT_COMPONENT
			}
		}

		if stateManifest != nil {
			if isDeploy {
				stateManifest = state.UpdateState(stateManifest, componentName,
					stackParameters, expandedComponentParameters,
					nil, allOutputs, stackManifest.Outputs,
					noEnvironmentProvides(provides),
					false)
			}
			status := fmt.Sprintf("%sing", request.Verb)
			stateManifest = state.UpdateComponentStatus(stateManifest, componentName, &componentManifest.Meta, status, "")
			stateManifest = state.UpdateStackStatus(stateManifest, status, "")
			stateManifest = state.UpdatePhase(stateManifest, operationLogId, componentName, "in-progress")
			stateUpdater(stateManifest)
			stateUpdater("sync")
		}

		randomSize := 128
		if opts := componentManifest.Lifecycle.Options; opts != nil {
			if rnd := opts.Random; rnd != nil && rnd.Bytes > 0 {
				randomSize = rnd.Bytes
			}
		}
		randomStr, random, err := util.Random(randomSize)
		if err != nil {
			util.Warn("Unable to set %s: %v", HubEnvVarNameRandom, err)
		}
		componentDir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)

		verb := maybeTestVerb(request.Verb, request.DryRun)

		preHookVerb := fmt.Sprintf("pre-%s", verb)
		stdout, stderr, err := fireHooks(preHookVerb, stackBaseDir, component, componentParameters, osEnv)
		if err != nil {
			if stateManifest != nil && request.WriteOplogToStateOnError {
				stateManifest = state.AppendOperationLog(stateManifest, operationLogId,
					fmt.Sprintf("%v%s", err, formatStdoutStderr(stdout, stderr)))
			}
			util.MaybeFatalf2(updateStateComponentFailed, "One of %s hooks failed. See logs", preHookVerb)
		}

		stdout, stderr, err = delegate(verb,
			component, componentManifest, componentParameters,
			componentDir, osEnv, randomStr, stackBaseDir)
		var rawOutputs parameters.RawOutputs
		if err != nil {
			if stateManifest != nil && request.WriteOplogToStateOnError {
				stateManifest = state.AppendOperationLog(stateManifest, operationLogId,
					fmt.Sprintf("%v%s", err, formatStdoutStderr(stdout, stderr)))
			}
			maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
				fmt.Sprintf("Component `%s` failed to %s: %v", componentName, request.Verb, err),
				updateStateComponentFailed)
			failedComponents = append(failedComponents, componentName)
		} else if isDeploy {
			rawOutputsCaptured, componentOutputs, dynamicProvides, errs := captureOutputs(componentName, componentDir, componentManifest, componentParameters,
				stdout, random)
			rawOutputs = rawOutputsCaptured
			if len(errs) > 0 {
				log.Printf("Component `%s` failed to %s", componentName, request.Verb)
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
					fmt.Sprintf("Component `%s` outputs capture failed:\n\t%s",
						componentName, util.Errors("\n\t", errs...)),
					updateStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
			}
			if len(componentOutputs) > 0 &&
				(config.Debug || (config.Verbose && len(request.Components) == 1)) {
				log.Print("Component outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
			parameters.MergeOutputs(allOutputs, componentOutputs)

			if request.GitOutputs {
				if config.Debug || (config.Verbose && request.GitOutputsStatus) {
					log.Print("Checking Git status")
				}
				git := gitOutputs(componentName, componentDir, request.GitOutputsStatus)
				if config.Debug && len(git) > 0 {
					log.Print("Implicit Git outputs added:")
					parameters.PrintCapturedOutputs(git)
				}
				parameters.MergeOutputs(allOutputs, git)
			}

			componentComplexOutputs := captureProvides(component, stackBaseDir, componentsBaseDir,
				componentManifest.Provides, componentOutputs)
			if len(componentComplexOutputs) > 0 &&
				(config.Debug || (config.Verbose && len(request.Components) == 1)) {
				log.Print("Component additional outputs captured:")
				parameters.PrintCapturedOutputs(componentComplexOutputs)
			}
			parameters.MergeOutputs(allOutputs, componentComplexOutputs)

			mergeProvides(provides, componentName, append(dynamicProvides, componentManifest.Provides...), componentOutputs)
		} else if isUndeploy {
			eraseProvides(provides, componentName)
			if stateManifest != nil {
				stateManifest.Provides = noEnvironmentProvides(provides)
			}
		}

		postHookVerb := fmt.Sprintf("post-%s", verb)
		stdout, stderr, err = fireHooks(postHookVerb, stackBaseDir, component, componentParameters, osEnv)
		if err != nil {
			if stateManifest != nil && request.WriteOplogToStateOnError {
				stateManifest = state.AppendOperationLog(stateManifest, operationLogId,
					fmt.Sprintf("%v%s", err, formatStdoutStderr(stdout, stderr)))
			}
			util.MaybeFatalf2(updateStateComponentFailed, "One of %s hooks failed. See logs", postHookVerb)
		}

		if ctx.Err() != nil {
			break
		}

		if stateManifest != nil && isDeploy {
			final := componentIndex == len(order)-1 || (len(request.Components) > 0 && request.LoadFinalState)
			stateManifest = state.UpdateState(stateManifest, componentName,
				stackParameters, expandedComponentParameters,
				rawOutputs, allOutputs, stackManifest.Outputs,
				noEnvironmentProvides(provides), final)
		}

		if err == nil && isDeploy {
			err = waitForReadyConditions(ctx, componentManifest.Lifecycle.ReadyConditions, componentParameters, allOutputs, component.Depends)
			if err != nil {
				log.Printf("Component `%s` failed to %s", componentName, request.Verb)
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
					fmt.Sprintf("Component `%s` ready condition failed: %v", componentName, err),
					updateStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
			}
		}

		if err == nil && config.Verbose {
			log.Printf("Component `%s` completed %s", componentName, request.Verb)
		}

		if stateManifest != nil {
			if !util.Contains(failedComponents, componentName) {
				stateManifest = state.UpdateComponentStatus(stateManifest, componentName, &componentManifest.Meta,
					fmt.Sprintf("%sed", request.Verb), "")
				stateManifest = state.UpdatePhase(stateManifest, operationLogId, componentName, "success")
				stateUpdater(stateManifest)
			}
		}

		// end of component cycle
	}

	stackReadyConditionFailed := false
	if isDeploy {
		err := waitForReadyConditions(ctx, stackManifest.Lifecycle.ReadyConditions, stackParameters, allOutputs, nil)
		if err != nil {
			message := fmt.Sprintf("Stack ready condition failed: %v", err)
			if stateManifest != nil {
				stateManifest = state.UpdateStackStatus(stateManifest, "incomplete", message)
				stateManifest = state.UpdateOperation(stateManifest, operationLogId, request.Verb, "error", nil)
				stateUpdater(stateManifest)
			}
			util.MaybeFatalf("%s", message)
			stackReadyConditionFailed = true
		}
	}

	if stateManifest != nil {
		if !stackReadyConditionFailed {
			status, message := calculateStackStatus(stackManifest, stateManifest, request.Verb)
			stateManifest = state.UpdateStackStatus(stateManifest, status, message)
			stateManifest = state.UpdateOperation(stateManifest, operationLogId, request.Verb, "success", nil)
			stateUpdater(stateManifest)
		}
		stateUpdater("sync")
	}

	var stackOutputs []parameters.ExpandedOutput
	if stateManifest != nil {
		stackOutputs = stateManifest.StackOutputs
	} else {
		stackOutputs = parameters.ExpandRequestedOutputs(stackParameters, allOutputs, stackManifest.Outputs, false)
	}

	if config.Verbose {
		if isDeploy {
			provides2 := noEnvironmentProvides(provides)
			if len(provides2) > 0 {
				log.Printf("%s provides:", strings.Title(stackManifest.Kind))
				util.PrintDeps(provides2)
			}
			printStackOutputs(stackOutputs)
		}
	}

	if config.Verbose {
		printEndBlurb(request, stackManifest)
	}
}

func optionalComponent(lifecycle *manifest.Lifecycle, componentName string) bool {
	return (len(lifecycle.Mandatory) > 0 && !util.Contains(lifecycle.Mandatory, componentName)) ||
		util.Contains(lifecycle.Optional, componentName)
}

func maybeFatalIfMandatory(lifecycle *manifest.Lifecycle, componentName string, msg string, cleanup func(string, bool)) {
	if optionalComponent(lifecycle, componentName) {
		util.Warn("%s", msg)
		if cleanup != nil {
			cleanup(msg, false)
		}
	} else {
		util.MaybeFatalf2(cleanup, "%s", msg)
	}
}

func addLockedParameter(params parameters.LockedParameters, name, env, value string) {
	if p, exist := params[name]; !exist || util.Empty(p.Value) {
		if exist && p.Env != "" {
			env = p.Env
		}
		if config.Debug {
			log.Printf("Adding implicit parameter %s = `%s` (env: %s)", name, value, env)
		}
		params[name] = parameters.LockedParameter{Name: name, Value: value, Env: env}
	}
}

func addLockedParameter2(params []parameters.LockedParameter, name, env, value string) []parameters.LockedParameter {
	for i := range params {
		p := &params[i]
		if p.Name == name {
			if p.Env == "" {
				p.Env = env
			}
			if util.Empty(p.Value) {
				p.Value = value
			}
			return params
		}
	}
	if config.Debug {
		log.Printf("Adding implicit parameter %s = `%s` (env: %s)", name, value, env)
	}
	return append(params, parameters.LockedParameter{Name: name, Value: value, Env: env})
}

func addHubProvides(params []parameters.LockedParameter, provides map[string][]string) []parameters.LockedParameter {
	return addLockedParameter2(params, "hub.provides", "HUB_PROVIDES", strings.Join(util.SortedKeys2(provides), " "))
}

func maybeTestVerb(verb string, test bool) string {
	if test {
		return verb + "-test"
	}
	return verb
}

func fireHooks(trigger string, stackBaseDir string, component *manifest.ComponentRef,
	componentParameters parameters.LockedParameters, osEnv []string,
) ([]byte, []byte, error) {
	hooks := findHooksByTrigger(trigger, component.Hooks)
	if len(hooks) == 0 {
		return nil, nil, nil
	}

	extensions := ext.GetExtensionLocations()
	paths := filepath.SplitList(os.Getenv("PATH"))

	searchDirs := []string{
		component.Source.Dir,
		stackBaseDir,
		filepath.Join(stackBaseDir, "bin"),
		filepath.Join(stackBaseDir, ".hub"),
		filepath.Join(stackBaseDir, ".hub", "bin"),
	}
	searchDirs = append(searchDirs, extensions...)
	searchDirs = append(searchDirs, paths...)

	for _, hook := range hooks {
		var err error
		script, err := findScript(hook.File, searchDirs...)
		if err != nil || script == "" {
			util.Warn("Unable to locate hook script `%s:` %v", hook.File, err)
			continue
		}
		log.Printf("Running %s script: %s", trigger, util.HighlightColor(hook.File))
		if config.Verbose && len(componentParameters) > 0 {
			log.Print("Environment:")
			parameters.PrintLockedParameters(componentParameters)
		}
		stdout, stderr, err := delegateHook(script, stackBaseDir, component, componentParameters, osEnv)
		if err != nil {
			if strings.Contains(err.Error(), "fork/exec : no such file or directory") {
				log.Printf("Error: file %s has not been found.", script)
				return stdout, stderr, err
			} else if hook.Error == "ignore" {
				log.Printf("Error ignored: %s", err.Error())
				continue
			}
			log.Printf("Error: %s", err.Error())
			return stdout, stderr, err
		}
	}

	return nil, nil, nil
}

func findHooksByTrigger(trigger string, hooks []manifest.Hook) []manifest.Hook {
	matches := make([]manifest.Hook, 0)
	for i := range hooks {
		hook := hooks[i]
		for t := range hook.Triggers {
			glob := hook.Triggers[t]
			matched, _ := filepath.Match(glob, trigger)
			if matched {
				matches = append(matches, hook)
				break
			}
		}
	}
	return matches
}

func findScript(script string, dirs ...string) (string, error) {
	var result string
	// special case for absolute filenames - just check if it exists
	if filepath.IsAbs(script) {
		dir, f := filepath.Split(script)
		found, err := probeScript(dir, f)
		if err != nil {
			return "", err
		}
		if found == "" {
			return "", os.ErrNotExist
		}
		result = filepath.Join(dir, found)
	} else {
		var search []string
		for _, dir := range dirs {
			search = append(search, filepath.Join(dir, script))
		}

		for _, path := range search {
			dir, f := filepath.Split(path)
			found, _ := probeScript(dir, f)
			if found != "" {
				result = filepath.Join(dir, found)
				break
			}
		}
	}
	if result == "" {
		return "", os.ErrNotExist
	}
	if !filepath.IsAbs(result) {
		var err error
		result, err = filepath.Abs(result)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}

func delegateHook(script string, stackDir string, component *manifest.ComponentRef, componentParameters parameters.LockedParameters, osEnv []string) ([]byte, []byte, error) {
	var err error
	componentDir := component.Source.Dir
	// components usually stored as relative paths
	if !filepath.IsAbs(componentDir) {
		componentDir = filepath.Join(stackDir, componentDir)
		componentDir, err = filepath.Abs(componentDir)
		if err != nil {
			return nil, nil, err
		}
	}

	processEnv := parametersInEnv(component, componentParameters, stackDir)
	command := &exec.Cmd{
		Path: script,
		Dir:  componentDir,
		Env:  mergeOsEnviron(osEnv, processEnv),
	}
	return execImplementation(command, false, true)
}

func delegate(verb string, component *manifest.ComponentRef, componentManifest *manifest.Manifest,
	componentParameters parameters.LockedParameters,
	dir string, osEnv []string, random string, baseDir string,
) ([]byte, []byte, error) {
	if config.Debug && len(componentParameters) > 0 {
		log.Print("Component parameters:")
		parameters.PrintLockedParameters(componentParameters)
	}

	componentName := manifest.ComponentQualifiedNameFromRef(component)
	errs := processTemplates(component, &componentManifest.Templates, componentParameters, nil, dir)
	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("Failed to process templates:\n\t%s", util.Errors("\n\t", errs...))
	}

	processEnv := parametersInEnv(component, componentParameters, baseDir)
	impl, err := findImplementation(dir, verb, componentManifest)
	if err != nil {
		if componentManifest.Lifecycle.Bare == "allow" {
			if config.Verbose {
				log.Printf("Skip `%s`: %v", componentName, err)
			}
			return nil, nil, nil
		}
		return nil, nil, err
	}
	skaffoldEnvironment := skaffoldEnv(impl, processEnv)
	impl.Env = mergeOsEnviron(osEnv, processEnv, randomEnv(random), skaffoldEnvironment)
	if config.Debug && len(processEnv) > 0 {
		log.Print("Component environment:")
		printEnvironment(processEnv)
		printEnvironment(skaffoldEnvironment)
		if config.Trace {
			log.Print("Full process environment:")
			printEnvironment(impl.Env)
		}
	}

	stdout, stderr, err := execImplementation(impl, false, true)
	return stdout, stderr, err
}

func randomEnv(random string) []string {
	if random == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s=%s", HubEnvVarNameRandom, random)}
}

func skaffoldEnv(impl *exec.Cmd, processEnv []string) []string {
	if len(impl.Args) > 0 && impl.Args[0] == "skaffold" {
		for _, envEntry := range processEnv {
			if strings.HasPrefix(envEntry, SkaffoldKubeContextEnvVarName+"=") {
				return nil
			}
		}
		for _, envEntry := range processEnv {
			for _, domainVar := range []string{"DOMAIN_NAME", "DOMAIN"} {
				if strings.HasPrefix(envEntry, domainVar+"=") {
					kv := strings.SplitN(envEntry, "=", 2)
					return []string{fmt.Sprintf("%s=%s", SkaffoldKubeContextEnvVarName, kv[1])}
				}
			}
		}
	}
	return nil
}

func parametersInEnv(component *manifest.ComponentRef, componentParameters parameters.LockedParameters, baseDir string) []string {
	envParameters := make([]string, 0)
	envSetBy := make(map[string]string)
	envValue := make(map[string]string)
	for _, parameter := range componentParameters {
		if parameter.Env == "" {
			continue
		}
		currentValue := strings.TrimSpace(util.MaybeJson(parameter.Value))
		envParameters = append(envParameters, fmt.Sprintf("%s=%s", parameter.Env, currentValue))
		name := parameter.QName()
		setBy, exist := envSetBy[parameter.Env]
		if exist {
			prevValue := envValue[parameter.Env]
			if prevValue != currentValue { /*||
				(!strings.HasPrefix(setBy, name+"|") && !strings.HasPrefix(name, setBy+"|")) {*/
				util.Warn("Env var `%s=%s` set by `%s` overriden by `%s` to `%s`",
					parameter.Env, prevValue, setBy, name, currentValue)
			}
		}
		envSetBy[parameter.Env] = name
		envValue[parameter.Env] = currentValue
	}

	envComponentName := "COMPONENT_NAME"
	if setBy, exist := envSetBy[envComponentName]; !exist {
		envParameters = append(envParameters, fmt.Sprintf("%s=%s", envComponentName, component.Name))
	} else if config.Debug && envValue[envComponentName] != component.Name {
		log.Printf("Component `%s` env var `%s=%s` set by `%s`",
			component.Name, envComponentName, envValue[envComponentName], setBy)
	}

	var componentDir string
	if filepath.IsAbs(component.Source.Dir) {
		componentDir = component.Source.Dir
	} else {
		componentDir = filepath.Join(baseDir, component.Source.Dir)
		if !filepath.IsAbs(componentDir) {
			t, err := filepath.Abs(componentDir)
			if err != nil {
				util.Warn("Unable to set absolute path for HUB_COMPONENT_DIR (%s): %v", component.Name, err)
				util.Warn("Falling back to %s directory: %s", component.Name, componentDir)
			} else {
				componentDir = t
			}
		}
	}

	if !filepath.IsAbs(baseDir) {
		t, err := filepath.Abs(componentDir)
		if err != nil {
			util.Warn("Unable to take absolute path for %s (%s): %v", HubEnvVarNameStackBasedir, baseDir, err)
		} else {
			baseDir = t
		}
	}

	// for `hub render`
	envParameters = append(envParameters, fmt.Sprintf("%s=%s", HubEnvVarNameComponentName, component.Name))
	envParameters = append(envParameters, fmt.Sprintf("%s=%s", HubEnvVarNameComponentDir, componentDir))
	envParameters = append(envParameters, fmt.Sprintf("%s=%s", HubEnvVarNameStackBasedir, baseDir))

	return mergeOsEnviron(envParameters) // sort
}
