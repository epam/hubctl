package lifecycle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/satori/go.uuid"

	"hub/api"
	"hub/config"
	"hub/kube"
	"hub/manifest"
	"hub/parameters"
	"hub/state"
	"hub/storage"
	"hub/util"
)

const HubEnvVarNameComponentName = "HUB_COMPONENT"

func Execute(request *Request) {
	isDeploy := strings.HasPrefix(request.Verb, "deploy")
	isUndeploy := strings.HasPrefix(request.Verb, "undeploy")
	isStateFile := len(request.StateFilenames) > 0
	isSomeComponents := len(request.Components) > 0 || request.OffsetComponent != ""

	stackManifest, componentsManifests, chosenManifestFilename, err := manifest.ParseManifest(request.ManifestFilenames)
	if err != nil {
		log.Fatalf("Unable to %s: %s", request.Verb, err)
	}

	environment, err := manifest.ParseKvList(request.EnvironmentOverrides)
	if err != nil {
		log.Fatalf("Unable to parse environment settings `%s`: %v", request.EnvironmentOverrides, err)
	}

	osEnv, err := initOsEnv(request.OsEnvironmentMode)
	if err != nil {
		log.Fatalf("Unable to parse OS environment setup: %v", err)
	}

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
	// TODO check only -c / -o components sources
	checkComponentsSourcesExist(components, stackBaseDir, componentsBaseDir)
	checkLifecycleOrder(components, stackManifest.Lifecycle)
	checkLifecycleVerbs(components, componentsManifests, stackManifest.Lifecycle.Verbs, stackBaseDir, componentsBaseDir)
	checkLifecycleRequires(components, stackManifest.Lifecycle.Requires)
	manifest.CheckComponentsExist(components, append(request.Components, request.OffsetComponent, request.LimitComponent)...)
	optionalRequires := parseRequiresTunning(stackManifest.Lifecycle.Requires)
	provides := checkRequires(stackManifest.Requires, optionalRequires)
	mergePlatformProvides(provides, stackManifest.Platform.Provides)
	if config.Debug && len(provides) > 0 {
		log.Print("Requirements provided by:")
		util.PrintDeps(provides)
	}

	stateFiles, errs := storage.Check(request.StateFilenames, "state")
	if len(errs) > 0 {
		util.MaybeFatalf("Unable to check state files: %s", util.Errors2(errs...))
	}
	storage.EnsureNoLockFiles(stateFiles)

	var stateManifest *state.StateManifest
	if isStateFile && (isUndeploy || isSomeComponents) { // skip state file at start of full deploy
		var err error
		stateManifest, err = state.ParseState(stateFiles)
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
		}
	}

	deploymentIdParameterName := "hub.deploymentId"
	deploymentId := ""
	if stateManifest != nil {
		for _, p := range stateManifest.StackParameters {
			if p.Name == deploymentIdParameterName {
				deploymentId = p.Value
				break
			}
		}
	}
	if deploymentId == "" {
		u, err := uuid.NewV4()
		if err != nil {
			log.Fatalf("Unable to generate `hub.deploymentId` v4 UUID: %v", err)
		}
		deploymentId = u.String()
	}
	plainStackNameParameterName := "hub.stackName"
	plainStackName := util.PlainName(stackManifest.Meta.Name)
	extraExpansionValues := []manifest.Parameter{
		{Name: deploymentIdParameterName, Value: deploymentId},
		{Name: plainStackNameParameterName, Value: plainStackName},
	}

	// TODO state file has user-level parameters for undeploy operation
	// should we just go with the state values if we cannot lock all parameters properly?
	stackParameters, errs := parameters.LockParameters(
		manifest.FlattenParameters(stackManifest.Parameters, chosenManifestFilename),
		extraExpansionValues,
		func(parameter *manifest.Parameter) error {
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
		}
	}
	addLockedParameter(stackParameters, deploymentIdParameterName, "DEPLOYMENT_ID", deploymentId)
	addLockedParameter(stackParameters, plainStackNameParameterName, "STACK_NAME", plainStackName)
	stackParametersNoLinks := parameters.ParametersWithoutLinks(stackParameters)

	order := stackManifest.Lifecycle.Order
	if stateManifest != nil {
		stateManifest.Lifecycle.Order = order
	}

	offsetGuessed := false
	if isUndeploy {
		order = util.Reverse(order)

		// on undeploy, guess which component failed to deploy and start undeploy from it
		if request.GuessComponent && isStateFile && stateManifest != nil && !isSomeComponents {
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

	failedComponents := make([]string, 0)

NEXT_COMPONENT:
	for componentIndex, componentName := range order {
		if (len(request.Components) > 0 && !util.Contains(request.Components, componentName)) ||
			(offsetComponentIndex >= 0 && componentIndex < offsetComponentIndex) ||
			(limitComponentIndex >= 0 && componentIndex > limitComponentIndex) {
			if config.Debug {
				log.Printf("Skip %s", componentName)
			}
			continue
		}

		if config.Verbose {
			log.Printf("%s ***%s*** (%d/%d)", request.Verb, componentName, componentIndex+1, len(components))
		}

		component := findComponentRef(components, componentName)
		componentManifest := findComponentManifest(component, componentsManifests)

		if stateManifest != nil && (componentIndex == offsetComponentIndex || len(request.Components) > 0) {
			if len(request.Components) > 0 {
				allOutputs = make(parameters.CapturedOutputs)
			}
			state.MergeParsedStateOutputs(stateManifest,
				componentName, component.Depends, stackManifest.Lifecycle.Order, isDeploy,
				allOutputs)
		}

		var writeStateComponentFailed func(string, bool)
		if isStateFile {
			writeStateComponentFailed = func(msg string, final bool) {
				stackStatus := "incomplete"
				stackMessage := msg
				if config.Force || optionalComponent(&stackManifest.Lifecycle, componentName) {
					stackStatus = ""
					stackMessage = ""
				}
				var write bool
				stateManifest, write = state.UpdateStatus(stateManifest,
					componentName, "error", msg, stackStatus, stackMessage)
				if final && write {
					state.WriteState(stateManifest, stateFiles)
				}
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
					writeStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
				continue NEXT_COMPONENT
			}
		}

		expandedComponentParameters, expansionErrs := parameters.ExpandParameters(componentName, component.Depends,
			stackParameters, allOutputs,
			manifest.FlattenParameters(componentManifest.Parameters, componentManifest.Meta.Name),
			environment)
		expandedComponentParameters = addHubProvides(expandedComponentParameters, provides)
		allParameters := parameters.MergeParameters(stackParametersNoLinks, expandedComponentParameters)
		optionalParametersFalse := calculateOptionalFalseParameters(componentName, allParameters, optionalRequires)
		if len(optionalParametersFalse) > 0 {
			log.Printf("Surprisingly, let skip `%s` due to optional parameter %v evaluated to false", componentName, optionalParametersFalse)

			continue NEXT_COMPONENT
		}
		if len(expansionErrs) > 0 {
			log.Printf("Component `%s` failed to %s", componentName, request.Verb)
			maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
				fmt.Sprintf("Component `%s` parameters expansion failed:\n\t%s",
					componentName, util.Errors("\n\t", expansionErrs...)),
				writeStateComponentFailed)
			failedComponents = append(failedComponents, componentName)
			continue NEXT_COMPONENT
		}

		var componentParameters parameters.LockedParameters
		if request.StrictParameters {
			componentParameters = parameters.MergeParameters(make(parameters.LockedParameters), expandedComponentParameters)
		} else {
			componentParameters = allParameters
		}

		if optionalNotProvided, err := prepareComponentRequires(provides, componentManifest, allParameters, allOutputs, optionalRequires); len(optionalNotProvided) > 0 || err != nil {
			if err != nil {
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName, fmt.Sprintf("%v", err), writeStateComponentFailed)
				continue NEXT_COMPONENT
			}
			log.Printf("Skip %s due to unsatisfied optional requirements %v", componentName, optionalNotProvided)
			// there will be a gap in state file but `deploy -c` will be able to find some state from
			// a preceding component
			continue NEXT_COMPONENT
		}

		if isStateFile {
			status := fmt.Sprintf("%sing", request.Verb)
			write := true
			if isDeploy {
				stateManifest = state.UpdateState(stateManifest,
					componentName, status, status,
					stackParameters, expandedComponentParameters,
					nil, allOutputs, stackManifest.Outputs,
					noEnvironmentProvides(provides),
					false)
			} else {
				stateManifest, write = state.UpdateStatus(stateManifest,
					componentName, status, "", status, "")
			}
			if write {
				state.WriteState(stateManifest, stateFiles)
			}
		}

		dir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)
		stdout, err := delegate(request.Verb, component, componentManifest, componentParameters,
			dir, osEnv, request.PipeOutputInRealtime)

		var rawOutputs parameters.RawOutputs
		if err != nil {
			maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
				fmt.Sprintf("Component `%s` failed to %s: %v", componentName, request.Verb, err),
				writeStateComponentFailed)
			failedComponents = append(failedComponents, componentName)
		} else if isDeploy {
			var componentOutputs parameters.CapturedOutputs
			var dynamicProvides []string
			var errs []error
			rawOutputs, componentOutputs, dynamicProvides, errs =
				captureOutputs(componentName, componentParameters, stdout, componentManifest.Outputs)

			if len(errs) > 0 {
				log.Printf("Component `%s` failed to %s", componentName, request.Verb)
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
					fmt.Sprintf("Component `%s` outputs capture failed:\n\t%s",
						componentName, util.Errors("\n\t", errs...)),
					writeStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
			}
			if len(componentOutputs) > 0 &&
				(config.Debug || (config.Verbose && len(request.Components) == 1)) {
				log.Print("Component outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
			parameters.MergeOutputs(allOutputs, componentOutputs)

			if request.GitOutputs {
				if config.Verbose {
					log.Print("Checking Git status")
				}
				git := gitOutputs(componentName, dir, request.GitOutputsStatus)
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
		}

		if isDeploy && isStateFile {
			final := componentIndex == len(order)-1 || (len(request.Components) > 0 && request.LoadFinalState)
			stateManifest = state.UpdateState(stateManifest,
				componentName, "", "",
				stackParameters, expandedComponentParameters,
				rawOutputs, allOutputs, stackManifest.Outputs,
				noEnvironmentProvides(provides), final)
		}

		if err == nil && isDeploy {
			err = waitForReadyConditions(componentManifest.Lifecycle.ReadyConditions, componentParameters, allOutputs)
			if err != nil {
				log.Printf("Component `%s` failed to %s", componentName, request.Verb)
				maybeFatalIfMandatory(&stackManifest.Lifecycle, componentName,
					fmt.Sprintf("Component `%s` ready condition failed: %v", componentName, err),
					writeStateComponentFailed)
				failedComponents = append(failedComponents, componentName)
			}
		}

		if err == nil && config.Verbose {
			log.Printf("Component `%s` completed %s", componentName, request.Verb)
		}

		if isStateFile {
			if !util.Contains(failedComponents, componentName) {
				stateManifest, _ = state.UpdateStatus(stateManifest,
					componentName, fmt.Sprintf("%sed", request.Verb), "", "", "")
			}
			state.WriteState(stateManifest, stateFiles)
		}
	}

	finalStateWritten := false
	if isDeploy {
		err := waitForReadyConditions(stackManifest.Lifecycle.ReadyConditions, stackParameters, allOutputs)
		if err != nil {
			message := fmt.Sprintf("Stack ready condition failed: %v", err)
			if isStateFile {
				stateManifest, _ = state.UpdateStatus(stateManifest,
					"", "", "", "incomplete", message)
				state.WriteState(stateManifest, stateFiles)
				finalStateWritten = true
			}
			util.MaybeFatalf("%s", message)
		}
	}
	if !finalStateWritten && isStateFile {
		status, message := calculateStackStatus(stackManifest, stateManifest)
		state.UpdateStatus(stateManifest, "", "", "", status, message)
		state.WriteState(stateManifest, stateFiles)
	}

	var stackOutputs []parameters.ExpandedOutput
	if stateManifest != nil {
		stackOutputs = stateManifest.StackOutputs
	} else {
		stackOutputs = parameters.ExpandRequestedOutputs(stackParameters, allOutputs, stackManifest.Outputs, false)
	}

	if config.Verbose {
		if isDeploy {
			if len(provides) > 0 {
				log.Printf("%s provides:", strings.Title(stackManifest.Kind))
				util.PrintDeps(noEnvironmentProvides(provides))
			}
			printExpandedOutputs(stackOutputs)
		}
	}

	if request.SaveStackInstanceOutputs && request.StackInstance != "" && len(stackOutputs) > 0 {
		patch := api.StackInstancePatch{
			Outputs:  api.TransformStackOutputsToApi(stackOutputs),
			Provides: noEnvironmentProvides(provides),
		}
		if config.Verbose {
			log.Print("Sending stack outputs to Control Plane")
			if config.Trace {
				printStackInstancePatch(patch)
			}
		}
		_, err := api.PatchStackInstance(request.StackInstance, patch)
		if err != nil {
			util.Warn("Unable to send stack outputs to Control Plane: %v", err)
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
	if p, exist := params[name]; !exist || p.Value == "" {
		if exist && p.Env != "" {
			env = p.Env
		}
		if config.Verbose {
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
			if p.Value == "" {
				p.Value = value
			}
			return params
		}
	}
	if config.Verbose {
		log.Printf("Adding implicit parameter %s = `%s` (env: %s)", name, value, env)
	}
	return append(params, parameters.LockedParameter{Name: name, Value: value, Env: env})
}

func addHubProvides(params []parameters.LockedParameter, provides map[string][]string) []parameters.LockedParameter {
	return addLockedParameter2(params, "hub.provides", "HUB_PROVIDES", strings.Join(util.SortedKeys2(provides), " "))
}

func findComponentRef(components []manifest.ComponentRef, componentName string) *manifest.ComponentRef {
	for i, component := range components {
		name := manifest.ComponentQualifiedNameFromRef(&component)
		if name == componentName {
			return &components[i]
		}
	}
	return nil
}

func findComponentManifest(component *manifest.ComponentRef, componentsManifests []manifest.Manifest) *manifest.Manifest {
	name := manifest.ComponentQualifiedNameFromRef(component)
	for _, componentsManifest := range componentsManifests {
		if name == manifest.ComponentQualifiedNameFromMeta(&componentsManifest.Meta) {
			return &componentsManifest
		}
	}
	return nil
}

func delegate(verb string, component *manifest.ComponentRef, componentManifest *manifest.Manifest,
	componentParameters parameters.LockedParameters,
	dir string, osEnv []string, pipeOutputInRealtime bool) (string, error) {

	if config.Debug && len(componentParameters) > 0 {
		log.Print("Component parameters:")
		parameters.PrintLockedParameters(componentParameters)
	}

	componentName := manifest.ComponentQualifiedNameFromRef(component)
	errs := processTemplates(component, &componentManifest.Templates, componentParameters, nil, dir)
	if len(errs) > 0 {
		return "", fmt.Errorf("Failed to process templates:\n\t%s", util.Errors("\n\t", errs...))
	}

	processEnv := parametersInEnv(componentName, componentParameters)
	impl, err := findImplementation(dir, verb)
	if err != nil {
		if componentManifest.Lifecycle.Bare == "allow" {
			if config.Verbose {
				log.Printf("Skip `%s`: %v", componentName, err)
			}
			return "", nil
		}
		return "", err
	}
	impl.Env = mergeOsEnviron(osEnv, processEnv)
	if config.Debug && len(processEnv) > 0 {
		log.Print("Component environment:")
		printEnvironment(processEnv)
		if config.Trace {
			log.Print("Full process environment:")
			printEnvironment(impl.Env)
		}
	}

	stdout, stderr, err := execImplementation(impl, pipeOutputInRealtime)
	if !pipeOutputInRealtime && (config.Trace || err != nil) {
		log.Printf("%s%s", formatStdout("stdout", stdout), formatStdout("stderr", stderr))
	}
	return stdout.String(), err
}

func parametersInEnv(componentName string, componentParameters parameters.LockedParameters) []string {
	envParameters := make([]string, 0)
	envSetBy := make(map[string]string)
	envValue := make(map[string]string)
	for _, parameter := range componentParameters {
		if parameter.Env == "" {
			continue
		}
		currentValue := strings.TrimSpace(parameter.Value)
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
		envParameters = append(envParameters, fmt.Sprintf("%s=%s", envComponentName, componentName))
	} else if config.Debug && envValue[envComponentName] != componentName {
		log.Printf("Component `%s` env var `%s=%s` set by `%s`",
			componentName, envComponentName, envValue[envComponentName], setBy)
	}
	// for `hub render`
	envParameters = append(envParameters, fmt.Sprintf("%s=%s", HubEnvVarNameComponentName, componentName))

	return mergeOsEnviron(envParameters) // sort
}

func goWait(routine func()) chan string {
	ch := make(chan string)
	wrapper := func() {
		routine()
		ch <- "done"
	}
	go wrapper()
	return ch
}

func execImplementation(impl *exec.Cmd, pipeOutputInRealtime bool) (bytes.Buffer, bytes.Buffer, error) {
	stderr, err := impl.StderrPipe()
	if err != nil {
		log.Fatalf("Unable to obtain stderr pipe: %v", err)
	}
	stdout, err := impl.StdoutPipe()
	if err != nil {
		log.Fatalf("Unable to obtain stdout pipe: %v", err)
	}
	os.Stdout.Sync()
	os.Stderr.Sync()

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	var stdoutWritter io.Writer = &stdoutBuffer
	var stderrWritter io.Writer = &stderrBuffer
	if pipeOutputInRealtime {
		stdoutWritter = io.MultiWriter(&stdoutBuffer, os.Stdout)
		stderrWritter = io.MultiWriter(&stderrBuffer, os.Stderr)
		args := ""
		if len(impl.Args) > 1 {
			args = fmt.Sprintf(" %v", impl.Args[1:])
		}
		fmt.Printf("--- %s%s (%s)\n", impl.Path, args, impl.Dir)
	}

	stdoutComplete := goWait(func() { io.Copy(stdoutWritter, stdout) })
	stderrComplete := goWait(func() { io.Copy(stderrWritter, stderr) })
	// Wait will close the pipe after seeing the command exit, so most callers
	// need not close the pipe themselves; however, an implication is that it is
	// incorrect to call Wait before all reads from the pipe have completed.
	// For the same reason, it is incorrect to call Run when using StdoutPipe.
	err = impl.Start()
	<-stdoutComplete
	<-stderrComplete

	if pipeOutputInRealtime {
		fmt.Print("---\n")
	}

	os.Stdout.Sync()
	os.Stderr.Sync()

	if err == nil {
		err = impl.Wait()
	}

	return stdoutBuffer, stderrBuffer, err
}

func captureOutputs(componentName string, componentParameters parameters.LockedParameters,
	textOutput string, requestedOutputs []manifest.Output) (parameters.RawOutputs, parameters.CapturedOutputs, []string, []error) {

	tfOutputs := parseTextOutput(textOutput)
	dynamicProvides := extractDynamicProvides(tfOutputs)
	kv := parameters.ParametersKV(componentParameters)

	outputs := make(parameters.CapturedOutputs)
	errs := make([]error, 0)
	for _, requestedOutput := range requestedOutputs {
		output := parameters.CapturedOutput{Component: componentName, Name: requestedOutput.Name, Kind: requestedOutput.Kind}
		if requestedOutput.FromTfVar != "" {
			variable, encoding := valueEncoding(requestedOutput.FromTfVar)
			value, exist := tfOutputs[variable]
			if !exist {
				errs = append(errs, fmt.Errorf("Unable to capture raw output `%s` for component `%s` output `%s`",
					variable, componentName, requestedOutput.Name))
				value = "(unknown)"
			}
			if exist && encoding != "" {
				if encoding == "base64" {
					bValue, err := base64.StdEncoding.DecodeString(value)
					if err != nil {
						errs = append(errs, fmt.Errorf("Unable to decode base64 `%s` while capturing output `%s` from raw output `%s`: %v",
							util.Trim(value), requestedOutput.FromTfVar, variable, err))
					} else {
						value = string(bValue)
					}
				} else {
					errs = append(errs, fmt.Errorf("Unknown encoding `%s` capturing output `%s` from raw output `%s`",
						encoding, requestedOutput.FromTfVar, variable))
				}
			}
			output.Value = value
		} else {
			if requestedOutput.Value == "" {
				requestedOutput.Value = fmt.Sprintf("${%s}", requestedOutput.Name)
			}
			if parameters.RequireExpansion(requestedOutput.Value) {
				value := parameters.CurlyReplacement.ReplaceAllStringFunc(requestedOutput.Value,
					func(variable string) string {
						variable = parameters.StripCurly(variable)
						substitution, exist := parameters.FindValue(variable, componentName, nil, kv)
						if !exist {
							errs = append(errs, fmt.Errorf("Component `%s` output `%s = %s` refer to unknown substitution `%s`",
								componentName, requestedOutput.Name, requestedOutput.Value, variable))
							substitution = "(unknown)"
						}
						if parameters.RequireExpansion(substitution) {
							errs = append(errs, fmt.Errorf("Component `%s` output `%s = %s` refer to substitution `%s` that expands to `%s`. This is surely a bug.",
								componentName, requestedOutput.Name, requestedOutput.Value, variable, substitution))
							substitution = "(bug)"
						}
						return substitution
					})
				output.Value = value
			} else {
				output.Value = requestedOutput.Value
			}
		}
		outputs[output.QName()] = output
		kv[requestedOutput.Name] = output.Value
	}
	if len(errs) > 0 {
		if len(tfOutputs) > 0 {
			log.Print("Raw outputs:")
			util.PrintMap(tfOutputs)
		} else {
			log.Print("No raw outputs captured")
		}
	}
	return tfOutputs, outputs, dynamicProvides, errs
}

func parseTextOutput(textOutput string) parameters.RawOutputs {
	outputs := make(map[string][]string)
	outputsMarker := "Outputs:\n"
	chunk := 1
	for {
		i := strings.Index(textOutput, outputsMarker)
		if i == -1 {
			if config.Debug && len(outputs) > 0 {
				log.Print("Parsed raw outputs:")
				util.PrintMap2(outputs)
			}
			rawOutputs := make(parameters.RawOutputs)
			for k, list := range outputs {
				rawOutputs[k] = strings.Join(list, ",")
			}
			return rawOutputs
		}
		markerFound := i == 0 || (i > 0 && textOutput[i-1] == '\n')
		textOutput = textOutput[i+len(outputsMarker):]
		if !markerFound {
			continue
		}
		textFragment := textOutput
		i = strings.Index(textFragment, "\n\n")
		if i > 0 {
			textFragment = textFragment[:i]
		}
		if config.Debug {
			log.Printf("Parsing output chunk #%d:\n%s", chunk, textFragment)
			chunk++
		}
		lines := strings.Split(textFragment, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "#") {
				continue
			}
			kv := strings.SplitN(line, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := util.TrimColor(util.Trim(kv[0]))
			value := util.TrimColor(util.Trim(kv[1]))
			// accumulate repeating keys
			list, exist := outputs[key]
			if exist {
				if !util.Contains(list, value) {
					outputs[key] = append(list, value)
				}
			} else {
				outputs[key] = []string{value}
			}
		}
	}
}

func extractDynamicProvides(rawOutputs parameters.RawOutputs) []string {
	key := "provides"
	if v, exist := rawOutputs[key]; exist {
		return strings.Split(v, ",")
	}
	return []string{}
}

func gitOutputs(componentName, dir string, status bool) parameters.CapturedOutputs {
	keys, err := gitStatus(dir, status)
	if err != nil {
		util.Warn("Unable to capture `%s` Git status: %v", componentName, err)
	}
	if len(keys) > 0 {
		base := fmt.Sprintf("hub.components.%s.git", componentName)
		outputs := make(parameters.CapturedOutputs)
		for k, v := range keys {
			outputName := fmt.Sprintf("%s.%s", base, k)
			outputs[outputName] = parameters.CapturedOutput{Component: componentName, Name: outputName, Value: v}
		}
		return outputs
	}
	return nil
}

func captureProvides(component *manifest.ComponentRef, stackBaseDir string, componentsBaseDir string, provides []string,
	componentOutputs parameters.CapturedOutputs) parameters.CapturedOutputs {

	outputs := make(parameters.CapturedOutputs)
	for _, prov := range provides {
		switch prov {
		case "kubernetes":
			kubernetesParams := kube.CaptureKubernetes(component, stackBaseDir, componentsBaseDir, componentOutputs)
			parameters.MergeOutputs(outputs, kubernetesParams)

		default:
		}
	}
	return outputs
}

func mergePlatformProvides(provides map[string][]string, platformProvides []string) {
	platform := "*platform*"
	for _, provide := range platformProvides {
		providers, exist := provides[provide]
		if exist {
			providers = append(providers, platform)
		} else {
			providers = []string{platform}
		}
		provides[provide] = providers
	}
}

func mergeProvides(provides map[string][]string, componentName string, componentProvides []string,
	componentOutputs parameters.CapturedOutputs) {

	for _, prov := range componentProvides {
		switch prov {
		case "kubernetes":
			for _, reqOutput := range []string{"dns.domain"} {
				qName := parameters.OutputQualifiedName(reqOutput, componentName)
				_, exist := componentOutputs[qName]
				if !exist {
					log.Printf("Component `%s` declared to provide `%s` but no `%s` output found",
						componentName, prov, qName)
					log.Print("Outputs:")
					parameters.PrintCapturedOutputs(componentOutputs)
					if !config.Force {
						os.Exit(1)
					}
				}
			}

		default:
		}

		who, exist := provides[prov]
		if !exist {
			who = []string{componentName}
		} else if !util.Contains(who, componentName) { // check because of re-deploy
			if config.Debug {
				log.Printf("`%s` already provides `%s`, but component `%s` also provides `%s`",
					strings.Join(who, ", "), prov, componentName, prov)
			}
			who = append(who, componentName)
		}
		provides[prov] = who
	}
}
