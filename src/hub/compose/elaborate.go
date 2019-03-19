package compose

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/kube"
	"hub/manifest"
	"hub/state"
	"hub/storage"
	"hub/util"
)

var globalEnvVarsAllowed = []string{
	"DEPLOYMENT_ID",
	"AWS_DEFAULT_REGION",
	"STATE_BUCKET",
	"STATE_REGION",
	"NAME",
	"BASE_DOMAIN",
}

// this must match to lifecycle.checkRequires()
var requirementProvidedByEnvironment = []string{
	"aws", "gcp", "gcs", "kubectl", "kubernetes", "helm", "vault",
}

var environment map[string]string

func Elaborate(manifestFilename string,
	parametersFilenames []string, environmentOverrides, explicitProvides string,
	stateManifests []string, elaborateManifests []string, componentsBaseDir string) {

	if config.Verbose {
		parametersFrom := ""
		if len(parametersFilenames) > 0 {
			parametersFrom = fmt.Sprintf(" with parameters from %s", strings.Join(parametersFilenames, ", "))
		}
		overrides := ""
		if environmentOverrides != "" {
			overrides = fmt.Sprintf(" with environment overrides: %s", environmentOverrides)
		}
		state := ""
		if len(stateManifests) > 0 {
			state = fmt.Sprintf(" with state outputs from %v", stateManifests)
		}
		log.Printf("Assembling %v from `%s`%s%s%s", elaborateManifests, manifestFilename,
			parametersFrom, overrides, state)
	}

	environment, err := manifest.ParseKvList(environmentOverrides)
	if err != nil {
		log.Fatalf("Unable to parse environment settings `%s`: %v", environmentOverrides, err)
	}

	wellKnown, err := manifest.GetWellKnownParametersManifest()
	if err != nil {
		log.Printf("No well-known parameters loaded: %v", err)
		wellKnown = &manifest.WellKnownParametersManifest{}
	}
	wellKnownKV := make(map[string]manifest.Parameter)
	for _, known := range wellKnown.Parameters {
		wellKnownKV[known.Name] = known
	}

	var st *state.StateManifest
	if len(stateManifests) > 0 {
		st = state.MustParseStateFiles(stateManifests)
	}

	stackManifest, componentsManifests := elaborate(manifestFilename, parametersFilenames, environment,
		wellKnownKV, componentsBaseDir, []string{}, 0)

	isApplication := stackManifest.Kind == "application"

	if isApplication {
		checkApplicationNameClash(stackManifest)
	}

	platformProvides := util.SplitPaths(explicitProvides)
	if len(platformProvides) > 0 {
		stackManifest.Requires = connectExplicitProvides(stackManifest.Requires, platformProvides)
		sort.Strings(platformProvides)
	}
	if st != nil {
		setValuesFromState(stackManifest.Parameters, st)
		stackManifest.Requires = connectStateProvides(stackManifest.Requires, st.Provides)
		platformProvides = util.MergeUnique(platformProvides, util.SortedKeys2(st.Provides))
	}
	if len(platformProvides) > 0 {
		stackManifest.Platform.Provides = util.MergeUnique(stackManifest.Platform.Provides, platformProvides)
	}
	warnNoValue(stackManifest.Parameters)
	warnFromEnvValueMismatch(stackManifest.Parameters)

	if isApplication {
		bare := stackManifest.Lifecycle.Bare
		if bare != "" && bare != "allow" {
			util.Warn("`lifecycle.bare` specify `%s` but the only value recognized is `allow`", bare)
		}
		componentsManifests = transformApplicationIntoComponent(stackManifest, componentsManifests)
	}

	err = writeStackManifest(elaborateManifests, stackManifest, componentsManifests)
	if err != nil {
		log.Fatalf("Unable to write: %v", err)
	}
}

func elaborate(manifestFilename string, parametersFilenames []string, overrides map[string]string,
	wellKnown map[string]manifest.Parameter, componentsBaseDir string,
	excludedComponents []string, depth int) (*manifest.Manifest, []manifest.Manifest) {

	stackManifest := parseManifest(manifestFilename)

	parametersManifests, parametersFilenamesRead := parseParameters(parametersFilenames)

	stackBaseDir := util.StripDotDirs(filepath.Dir(manifestFilename))
	componentsBaseDirCurrent := componentsBaseDir
	if componentsBaseDirCurrent == "" {
		componentsBaseDirCurrent = stackBaseDir
	}
	if config.Debug {
		log.Printf("Base directory for sources is `%s`", componentsBaseDirCurrent)
	}

	componentsManifests, err := manifest.ParseComponentsManifestsWithExclusion(stackManifest.Components, excludedComponents,
		stackBaseDir, componentsBaseDirCurrent)
	if err != nil {
		log.Fatalf("Unable to load component manifest refered from `%s`: %v", manifestFilename, err)
	}

	checkComponentsNames(stackManifest.Components)
	checkLifecycle(stackManifest.Components, stackManifest.Lifecycle)

	isApplication := stackManifest.Kind == "application"

	fromStack := stackManifest.Meta.FromStack != ""
	fromStackName := ""
	fromStackManifest := &manifest.Manifest{}
	var fromStackComponentsManifests []manifest.Manifest

	if fromStack {
		if isApplication {
			log.Fatalf("Application manifest %s cannot use `fromStack`", manifestFilename)
		}
		fromStackName = filepath.Base(stackManifest.Meta.FromStack)
		fromStackFilename := filepath.Join(stackManifest.Meta.FromStack, "hub.yaml")
		fromStackParams := scanParamsFiles(stackManifest.Meta.FromStack)
		fromStackExcludedComponents := append(excludedComponents, manifest.ComponentsNamesFromRefs(stackManifest.Components)...)
		fromStackManifest, fromStackComponentsManifests = elaborate(fromStackFilename, fromStackParams, overrides,
			wellKnown, componentsBaseDir, fromStackExcludedComponents, depth+1)
	}

	if config.Verbose {
		components := "with no sub-components"
		if len(stackManifest.Components) > 0 {
			components = fmt.Sprintf("with components: %s",
				strings.Join(manifest.ComponentsNamesFromRefs(stackManifest.Components), ", "))
		}
		log.Printf("*** %s %s %s", strings.Title(stackManifest.Kind), stackManifest.Meta.Name,
			components)
	}

	parameters := unwrapComponentsParameters(componentsManifests)
	checkParameters(parameters)
	if fromStack {
		parameters = append(parameters, fromStackManifest.Parameters) // already flat
	}
	manifestsParameters := [][]manifest.Parameter{
		manifest.FlattenParameters(stackManifest.Parameters, fmt.Sprintf("%s [%s]", stackManifest.Meta.Name, manifestFilename)),
	}
	manifestsParameters = append(manifestsParameters, unwrapManifestsParameters(parametersManifests, parametersFilenamesRead)...)
	checkParameters(manifestsParameters)

	var elaborated manifest.Manifest

	nComponents := len(componentsManifests)
	elaborated.Version = stackManifest.Version
	elaborated.Kind = stackManifest.Kind
	elaborated.Meta = stackManifest.Meta
	elaborated.Meta.FromStack = ""
	if fromStack {
		elaborated.Meta.Annotations = mergeAnnotations(fromStackManifest.Meta.Annotations, stackManifest.Meta.Annotations)
		parentBaseDir := stackManifest.Meta.FromStack
		parentComponentsBaseDir := componentsBaseDir
		if parentComponentsBaseDir == "" {
			parentComponentsBaseDir = parentBaseDir
		}
		elaborated.Components = mergeComponentsRefs(parentBaseDir, parentComponentsBaseDir,
			fromStackManifest.Components, stackManifest.Components)
		elaborated.Lifecycle = mergeLifecycle(fromStackManifest.Lifecycle, stackManifest.Lifecycle)
		elaborated.Outputs = mergeOutputs(fromStackManifest.Outputs, stackManifest.Outputs)
		componentsManifests = mergeComponentsManifests(fromStackComponentsManifests, componentsManifests)
		elaborated.Platform.Provides = util.MergeUnique(fromStackManifest.Platform.Provides, stackManifest.Platform.Provides)
	} else {
		elaborated.Components = stackManifest.Components
		elaborated.Lifecycle = stackManifest.Lifecycle
		elaborated.Outputs = stackManifest.Outputs
		elaborated.Platform.Provides = stackManifest.Platform.Provides
	}
	parametersManifestsOutputs := unwrapManifestsOutputs(parametersManifests)
	if len(parametersManifestsOutputs) > 0 {
		elaborated.Outputs = mergeOutputs(elaborated.Outputs, parametersManifestsOutputs)
	}
	if isApplication {
		elaborated.Templates = stackManifest.Templates
	}
	stackRequires := connectRequires(fromStackName, fromStackManifest.Provides,
		stackManifest.Requires, componentsManifests, stackManifest.Lifecycle.Order)
	elaborated.Requires = mergeRequires(fromStackManifest.Requires, stackRequires)
	elaborated.Provides = mergeProvides(fromStackName, fromStackManifest.Provides,
		stackManifest.Provides, componentsManifests)

	if depth == 0 && util.ContainsAny(elaborated.Requires, []string{"kubernetes", "kubectl"}) {
		// TODO distinguish EKS/etc. by kubernetes.flavor and setup appropriately
		parameters = append(parameters, manifest.ParameterWrap(kube.RequiredKubernetesParameters()))
	}
	parameters = append(parameters, manifestsParameters...)
	elaborated.Parameters = mergeParameters(parameters, overrides, wellKnown,
		manifest.ComponentsNamesFromRefs(elaborated.Components), nComponents, isApplication)

	return &elaborated, componentsManifests
}

func parseManifest(manifestFilename string) *manifest.Manifest {
	stackManifest, rest, _, err := manifest.ParseManifest([]string{manifestFilename})
	if err != nil {
		log.Fatalf("Unable to elaborate %s: %v", manifestFilename, err)
	}
	if len(rest) > 0 {
		util.Warn("Manifest %s contains multiple YAML documents - using first document only", manifestFilename)
	}
	allowedKinds := []string{"stack", "application"}
	if !util.Contains(allowedKinds, stackManifest.Kind) {
		util.Warn("Manifest `kind` must be one of %v, found `%s`", allowedKinds, stackManifest.Kind)
	}
	return stackManifest
}

func parseParameters(parametersFilenames []string) ([]*manifest.ParametersManifest, []string) {
	parametersManifests := make([]*manifest.ParametersManifest, 0, len(parametersFilenames))
	parametersFilenamesRead := make([]string, 0, len(parametersFilenames))
	for _, parametersFilename := range parametersFilenames {
		parametersManifest, parametersFilenameRead, err := manifest.ParseParametersManifest(
			util.SplitPaths(parametersFilename))
		if err != nil {
			log.Fatalf("Unable to load parameters %s: %v", parametersFilename, err)
		}
		parametersManifests = append(parametersManifests, parametersManifest)
		parametersFilenamesRead = append(parametersFilenamesRead, parametersFilenameRead)
	}
	return parametersManifests, parametersFilenamesRead
}

func scanParamsFiles(baseDir string) []string {
	params := []string{"params.yaml"}
	env := os.Getenv("ENV")
	if env != "" {
		params = append(params, fmt.Sprintf("params-%s.yaml", env))
	}
	exists := make([]string, 0, len(params))
	for _, filename := range params {
		path := filepath.Join(baseDir, filename)
		_, err := os.Stat(path)
		if err != nil {
			if !util.NoSuchFile(err) {
				log.Fatalf("Unable to stat `%s`: %v", path, err)
			}
		} else {
			exists = append(exists, path)
		}
	}
	return exists
}

func nameWithoutVersion(name string) string {
	if i := strings.Index(name, ":"); i > 0 {
		return name[:i]
	}
	return name
}

func checkApplicationNameClash(manifest *manifest.Manifest) {
	name := nameWithoutVersion(manifest.Meta.Name)
	if util.Contains(manifest.Lifecycle.Order, name) {
		log.Fatalf("Application name `%s` cannot clash with component name", name)
	}
}

func checkComponentsNames(componentsManifests []manifest.ComponentRef) {
	components := make(map[string]bool)
	for _, component := range componentsManifests {
		name := manifest.ComponentQualifiedNameFromRef(&component)
		_, exist := components[name]
		if exist {
			log.Fatalf("Duplicate component name `%s` ", name)
		}
		components[name] = true
	}
}

func checkLifecycle(components []manifest.ComponentRef, lifecycle manifest.Lifecycle) {
	refs := manifest.ComponentsNamesFromRefs(components)
	sorted := make([]string, len(refs))
	copy(sorted, refs)
	order := make([]string, len(lifecycle.Order))
	copy(order, lifecycle.Order)
	sort.Strings(sorted)
	sort.Strings(order)
	if !reflect.DeepEqual(sorted, order) {
		lifecycleOrder := "(not specified)"
		if len(lifecycle.Order) > 0 {
			lifecycleOrder = strings.Join(lifecycle.Order, ", ")
		}
		log.Fatalf("Components: %s;\n\tdoes not match deployment order: %s",
			strings.Join(refs, ", "), lifecycleOrder)
	}
	// not checking Mandatory and Optional as they could contain components from parent stack
}

func checkParameters(parametersAssorti [][]manifest.Parameter) {
	for _, parameters := range parametersAssorti {
		for _, parameter := range parameters {
			fqName := manifestParameterQualifiedName(&parameter)

			if parameter.Kind != "" && !util.Contains([]string{"user", "tech", "link"}, parameter.Kind) {
				util.Warn("Parameter `%s` specify unknown `kind: %s`",
					fqName, parameter.Kind)
			}
			if parameter.Kind == "link" && parameter.Value == "" {
				util.Warn("Parameter `%s` of kind `link` has no value assigned",
					fqName)
			}
			if parameter.Empty != "" && parameter.Empty != "allow" {
				util.Warn("Parameter `%s` specify `empty: %s` but the only value recognized is `allow`",
					fqName, parameter.Empty)
			}
		}
	}
}

func setValuesFromState(parameters []manifest.Parameter, st *state.StateManifest) {
	stateStackOutputs := make(map[string]string)

	// should we rely on explicit stack outputs only?
	for _, parameter := range st.StackParameters {
		// should we filter out `link` parameters?
		if parameter.Component == "" && parameter.Value != "" {
			stateStackOutputs[parameter.Name] = parameter.Value
		}
	}
	for _, output := range st.StackOutputs {
		name := output.Name
		if i := strings.Index(name, ":"); i > 0 && i < len(name)-1 {
			name = name[i+1:]
		}
		stateStackOutputs[name] = output.Value
	}

	for i, _ := range parameters {
		parameter := &parameters[i]
		if strings.HasPrefix(parameter.Name, "hub.") {
			continue
		}
		if parameter.Value == "" {
			value, exist := stateStackOutputs[parameter.Name]
			if exist {
				if parameter.Kind == "user" && parameter.Default == "" {
					parameter.Default = value
				} else {
					parameter.Value = value
				}
			} else {
				// a special case for Kubernetes keys
				if strings.HasPrefix(parameter.Name, "kubernetes.") {
					for _, providerName := range kube.KubernetesDefaultProviders {
						provider, exist := st.Components[providerName]
						if exist {
							for _, output := range provider.CapturedOutputs {
								if output.Name == parameter.Name {
									parameter.Value = output.Value
									break
								}
							}
							break
						}
					}
				}
			}
		}
	}
}

func warnNoValue(parameters []manifest.Parameter) {
	for _, parameter := range parameters {
		if parameter.Value == "" {
			who := "Parameter"
			noDefault := ""
			if parameter.Kind == "user" {
				if parameter.Default != "" || parameter.FromEnv != "" {
					continue
				}
				who = "User-level parameter"
				noDefault = " nor default"
			}
			util.Warn("%s `%s` has no value%s assigned",
				who, manifestParameterQualifiedName(&parameter), noDefault)
		}
	}
}

func warnFromEnvValueMismatch(parameters []manifest.Parameter) {
	for _, parameter := range parameters {
		if parameter.Kind == "user" && parameter.FromEnv != "" && parameter.Value != "" {
			if value, exist := os.LookupEnv(parameter.FromEnv); exist && value != parameter.Value {
				util.Warn("Parameter `%s` value `%s` differs from value `%s` provided by `fromEnv:` environment variable `%s`",
					manifestParameterQualifiedName(&parameter), parameter.Value, value, parameter.FromEnv)
			}
		}
	}
}

func transformApplicationIntoComponent(stack *manifest.Manifest, components []manifest.Manifest) []manifest.Manifest {
	name := nameWithoutVersion(stack.Meta.Name)

	stack.Lifecycle.Order = append(stack.Lifecycle.Order, name)

	applicationRef := manifest.ComponentRef{
		Name:   name,
		Source: stack.Meta.Source,
	}
	stack.Components = append(stack.Components, applicationRef)

	componentOutputs := make([]manifest.Output, 0, len(stack.Outputs))
	stackOutputs := make([]manifest.Output, 0, len(stack.Outputs))
	for _, output := range stack.Outputs {
		if output.Value != "" || output.FromTfVar != "" {
			componentOutputs = append(componentOutputs, output)
			stackOutput := manifest.Output{
				Name:        output.Name,
				Value:       fmt.Sprintf("${%s:%s}", name, output.Name),
				Brief:       output.Brief,
				Description: output.Description,
			}
			stackOutputs = append(stackOutputs, stackOutput)
		} else {
			stackOutputs = append(stackOutputs, output)
		}
	}

	componentParameters := make([]manifest.Parameter, 0, len(stack.Parameters))
	stackParameters := make([]manifest.Parameter, 0, len(stack.Parameters))
	for _, param := range stack.Parameters {
		if param.Component == "" || param.Component == name {
			componentParameters = append(componentParameters,
				manifest.Parameter{
					Name: param.Name,
					Env:  param.Env,
				})
		}
		param.Env = ""
		stackParameters = append(stackParameters, param)
	}

	componentManifest := manifest.Manifest{
		Version: stack.Version,
		Kind:    "component",
		Meta:    stack.Meta,
		Lifecycle: manifest.Lifecycle{
			Bare:            stack.Lifecycle.Bare,
			Verbs:           stack.Lifecycle.Verbs,
			ReadyConditions: stack.Lifecycle.ReadyConditions,
			Requires:        stack.Lifecycle.Requires,
		},
		Provides:   stack.Provides,
		Requires:   stack.Requires,
		Parameters: componentParameters,
		Templates:  stack.Templates,
		Outputs:    componentOutputs,
	}
	componentManifest.Meta.Name = name
	components = append(components, componentManifest)

	stack.Outputs = stackOutputs
	stack.Parameters = stackParameters
	stack.Templates = manifest.TemplateSetup{}

	return components
}

func manifestParameterQualifiedName(parameter *manifest.Parameter) string {
	return parameterQualifiedName(parameter.Name, parameter.Component)
}

func parameterQualifiedName(name string, component string) string {
	if component != "" {
		return fmt.Sprintf("%s|%s", name, component)
	}
	return name
}

func globalEnvVarAllowed(name string) bool {
	for _, allowed := range globalEnvVarsAllowed {
		if name == allowed {
			return true
		}
	}
	return false
}

func unwrapComponentsParameters(componentsManifests []manifest.Manifest) [][]manifest.Parameter {
	parameters := make([][]manifest.Parameter, 0, len(componentsManifests))
	for _, componentsManifest := range componentsManifests {
		for i, _ := range componentsManifest.Parameters {
			componentsManifest.Parameters[i].Component = componentsManifest.Meta.Name
		}
		parameters = append(parameters, manifest.FlattenParameters(componentsManifest.Parameters, componentsManifest.Meta.Name))
	}
	return parameters
}

func unwrapManifestsParameters(parametersManifests []*manifest.ParametersManifest, parametersFilenames []string) [][]manifest.Parameter {
	parameters := make([][]manifest.Parameter, 0, len(parametersManifests))
	for i, parametersManifest := range parametersManifests {
		parameters = append(parameters, manifest.FlattenParameters(parametersManifest.Parameters, parametersFilenames[i]))
	}
	return parameters
}

func unwrapManifestsOutputs(parametersManifests []*manifest.ParametersManifest) []manifest.Output {
	outputs := make([]manifest.Output, 0)
	for _, parametersManifest := range parametersManifests {
		outputs = append(outputs, parametersManifest.Outputs...)
	}
	return outputs
}

func mergeAnnotations(parent, child map[string]string) map[string]string {
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}
	if len(parent) == 0 && len(child) != 0 {
		return child
	}
	if len(parent) != 0 && len(child) == 0 {
		return parent
	}
	merged := make(map[string]string)
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

var envWarningsEmited = make(map[string]struct{})

func mergeParameters(parametersAssorti [][]manifest.Parameter,
	overrides map[string]string,
	wellKnown map[string]manifest.Parameter,
	allComponentsNames []string, nComponents int,
	isApplication bool) []manifest.Parameter {

	kv := make(map[string]manifest.Parameter)
	for docIndex, parameters := range parametersAssorti {
		for _, parameter := range parameters {
			parameter = enrichParameter(parameter, wellKnown)
			fqName := manifestParameterQualifiedName(&parameter)

			if parameter.FromEnv != "" {
				if parameter.Kind == "" {
					parameter.Kind = "user"
				}
				if docIndex < nComponents {
					util.Warn("Parameter `%s` specify `fromEnv: %s` on hub-component.yaml level",
						fqName, parameter.FromEnv)
				}
			}

			if docIndex < nComponents {
				if parameter.Kind == "link" {
					util.Warn("Parameter `%s` specify `kind: link` on hub-component.yaml level - this is not supported",
						fqName)
				}
				if parameter.Kind != "user" && parameter.Value == "" && parameter.Default != "" {
					util.Warn("Parameter `%s` specify `default:` on hub-component.yaml level - use `value:` instead",
						fqName)
				}
				// parameters from Stack Manifest and Parameters files are a special treat
				// Component parameter is propagated to Stack Manifest only for kind == user
				if parameter.Kind != "user" {
					continue
				}
			}

			if parameter.Env != "" && !globalEnvVarAllowed(parameter.Env) {
				_, emitted := envWarningsEmited[fqName]
				if docIndex >= nComponents {
					if !emitted && !isApplication {
						util.Warn("Parameter `%s` specify `env: %s` on hub.yaml / params.yaml level",
							fqName, parameter.Env)
						envWarningsEmited[fqName] = struct{}{}
					}
				} else {
					if !emitted && config.Verbose {
						log.Printf("User-level parameter `%s` specify `env: %s` on hub-component.yaml level - not propagated to global env",
							fqName, parameter.Env)
						envWarningsEmited[fqName] = struct{}{}
					}
					parameter.Env = ""
				}
			}

			if parameter.Component == "" {
				fqNames := make([]string, 0, 1+len(allComponentsNames))
				fqNames = append(fqNames, parameter.Name)
				for _, componentName := range allComponentsNames {
					fqNames = append(fqNames, fmt.Sprintf("%s|%s", parameter.Name, componentName))
				}
				for i, fqName := range fqNames {
					p, exist := kv[fqName]
					if !exist {
						if i == 0 { // plain parameter name
							kv[fqName] = parameter
						}
					} else {
						kv[fqName] = mergeParameter(p, parameter, overrides, false)
					}
				}
			} else {
				fqName := fmt.Sprintf("%s|%s", parameter.Name, parameter.Component)
				p, exist := kv[fqName]
				if !exist {
					kv[fqName] = parameter
				} else {
					kv[fqName] = mergeParameter(p, parameter, overrides, false)
				}
			}
		}
	}

	return sortedParameters(kv)
}

func enrichParameter(parameter manifest.Parameter, wellKnownKV map[string]manifest.Parameter) manifest.Parameter {
	wellKnown, exist := wellKnownKV[parameter.Name]
	if !exist {
		return parameter
	}
	return mergeParameter(wellKnown, parameter, nil, true)
}

func sortedParameters(kv map[string]manifest.Parameter) []manifest.Parameter {
	names := make([]string, 0, len(kv))
	for name, _ := range kv {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]manifest.Parameter, 0, len(names))
	for _, name := range names {
		out = append(out, kv[name])
	}
	return out
}

func mergeParameter(base, over manifest.Parameter, overrides map[string]string,
	enrichment bool) manifest.Parameter {

	if base.Name != over.Name {
		log.Fatalf("Unable to merge parameters: `name` didn't match\n%+v\n%+v", base, over)
	}
	// if !(base.Kind == over.Kind || (base.Kind == "tech" && over.Kind == "") || (base.Kind == "" && over.Kind == "tech")) {
	//	log.Fatalf("Unable to merge parameters: `kind` didn't match:\n\tfrom: %+v\n\tinto: %+v", base, over)
	// }
	kind := base.Kind
	if over.Kind != "" {
		if kind == "" || (kind == "tech" && over.Kind == "user") || enrichment {
			kind = over.Kind
		}
	}
	brief := mergeField(base.Brief, over.Brief)
	description := mergeField(base.Description, over.Description)
	env := mergeField(base.Env, over.Env)
	fromEnv := mergeField(base.FromEnv, over.FromEnv)
	defaultValue := mergeField(base.Default, over.Default)
	value := mergeField(base.Value, over.Value)
	if fromEnv != "" && overrides != nil {
		envValue, exist := overrides[fromEnv]
		if exist {
			value = envValue
		}
	}
	empty := mergeField(base.Empty, over.Empty)
	if value != "" {
		empty = ""
	}
	merged := manifest.Parameter{Name: over.Name, Component: base.Component, Kind: kind,
		Brief: brief, Description: description,
		Default: defaultValue, Env: env, FromEnv: fromEnv, Value: value, Empty: empty}
	if config.Trace {
		log.Printf("Parameters merged:\n\t--- %+v\n\t+++ %+v\n\t=== %+v", base, over, merged)
	}
	return merged
}

func mergeField(base string, over string) string {
	out := base
	if over != "" {
		out = over
	}
	return out
}

func connectRequires(parentStackName string, parentStackProvides []string,
	stackRequires []string, componentsManifests []manifest.Manifest, order []string) []string {

	provides := make(map[string][]string)
	addProv := func(name, prov string) {
		who, exist := provides[prov]
		if !exist {
			provides[prov] = []string{name}
		} else {
			if config.Trace && (!strings.HasPrefix(who[0], "*") || len(who) > 1) {
				log.Printf("`%s` already provides `%s`, but component `%s` also provides `%s`",
					strings.Join(who, ", "), prov, name, prov)
			}
			provides[prov] = append(who, name)
		}
	}

	requires := make(map[string][]string)
	addReq := func(name, req string) {
		by, exist := provides[req]
		if exist {
			if config.Debug {
				log.Printf("Component `%s` requirement `%s` provided by `%s`",
					name, req, strings.Join(by, ", "))
			}
			return
		}
		who, exist := requires[req]
		if !exist {
			requires[req] = []string{name}
		} else {
			requires[req] = append(who, name)
		}
	}

	parentStack := fmt.Sprintf("*%s*", parentStackName)
	for _, parentProvide := range parentStackProvides {
		addProv(parentStack, parentProvide)
		if parentProvide == "kubernetes" {
			addProv(parentStack, "kubectl")
		}
	}

	stack := "*stack*"
	for _, req := range stackRequires {
		addReq(stack, req)
	}

	components := make(map[string]manifest.Manifest)
	for _, component := range componentsManifests {
		name := manifest.ComponentQualifiedNameFromMeta(&component.Meta)
		components[name] = component
	}
	for _, name := range order {
		component := components[name]
		for _, req := range component.Requires {
			addReq(name, req)
		}
		for _, prov := range component.Provides {
			addProv(name, prov)
			if prov == "kubernetes" {
				addProv(name, "kubectl")
			}
		}
	}

	if config.Debug && len(requires) > 0 {
		log.Print("Stack requires:")
		util.PrintDeps(requires)
	}
	keys := make([]string, 0, len(requires))
	for k := range requires {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func connectExplicitProvides(requires []string, provides []string) []string {
	genuine := make([]string, 0, len(requires))
	for _, r := range requires {
		if util.Contains(requirementProvidedByEnvironment, r) || !util.Contains(provides, r) {
			genuine = append(genuine, r)
		}
	}
	return genuine
}

func connectStateProvides(requires []string, provides map[string][]string) []string {
	genuine := make([]string, 0, len(requires))
	for _, r := range requires {
		if !util.Contains(requirementProvidedByEnvironment, r) {
			if providers, exist := provides[r]; exist {
				if !util.Contains(providers, "*environment*") {
					continue
				}
			}
		}
		genuine = append(genuine, r)
	}
	return genuine
}

func mergeRequires(parentStackRequires []string, stackRequires []string) []string {
	return util.MergeUnique(parentStackRequires, stackRequires)
}

func mergeProvides(parentStackName string, parentProvides []string,
	stackProvides []string, componentsManifests []manifest.Manifest) []string {

	provides := make(map[string][]string)
	add := func(name, prov string) {
		who, exist := provides[prov]
		if !exist {
			provides[prov] = []string{name}
		} else {
			if config.Trace && (!strings.HasPrefix(who[0], "*") || len(who) > 1) {
				log.Printf("`%s` already provides `%s`, but component `%s` also provides `%s`",
					strings.Join(who, ", "), prov, name, prov)
			}
			provides[prov] = append(who, name)
		}
	}

	parentStack := fmt.Sprintf("*%s*", parentStackName)
	for _, prov := range parentProvides {
		add(parentStack, prov)
	}

	stack := "*stack*"
	for _, prov := range stackProvides {
		add(stack, prov)
	}

	for _, component := range componentsManifests {
		name := manifest.ComponentQualifiedNameFromMeta(&component.Meta)
		for _, prov := range component.Provides {
			add(name, prov)
		}
	}

	if config.Debug && len(provides) > 0 {
		log.Print("Stack provides:")
		util.PrintDeps(provides)
	}
	keys := make([]string, 0, len(provides))
	for k := range provides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func mergeLifecycle(parent, child manifest.Lifecycle) manifest.Lifecycle {
	return manifest.Lifecycle{
		Bare:            util.Value(parent.Bare, child.Bare),
		Order:           mergeOrder(parent.Order, child.Order),
		ReadyConditions: mergeReadyCondition(parent.ReadyConditions, child.ReadyConditions),
		Verbs:           util.MergeUnique(parent.Verbs, child.Verbs),
		Mandatory:       util.MergeUnique(parent.Mandatory, child.Mandatory),
		Optional:        util.MergeUnique(parent.Optional, child.Optional),
		Requires:        mergeRequiresTuning(parent.Requires, child.Requires),
	}
}

func mergeOrder(parent, child []string) []string {
	overridesFromChild := make([]int, 0, len(child))
	overridesToParent := make([]int, 0, len(child))
	prevOverridesToParent := 0
	for i, component := range child {
		for j, exist := range parent {
			if component == exist {
				if j < prevOverridesToParent {
					log.Fatalf("Component `%s` must come after `%s` in child stack `lifecycle.order` - as defined by parent stack (fromStack)",
						parent[prevOverridesToParent], component)
				}
				prevOverridesToParent = j
				overridesFromChild = append(overridesFromChild, i)
				overridesToParent = append(overridesToParent, j)
				break
			}
		}
	}

	order := make([]string, 0, len(parent)+len(child))
	if len(overridesFromChild) == 0 {
		order = append(order, parent...)
		order = append(order, child...)
	} else {
		relative := func(indices []int) {
			prev := 0
			for i, index := range indices {
				indices[i] = index - prev - i
				prev = index
			}
		}

		relative(overridesToParent)
		relative(overridesFromChild)

		parentBlocks := make([][]string, 0, len(child))
		for _, cutAt := range overridesToParent {
			parentBlocks = append(parentBlocks, parent[:cutAt])
			if cutAt == len(parent)-1 {
				parent = []string{}
			} else {
				parent = parent[cutAt+1:]
			}
		}
		parentBlocks = append(parentBlocks, parent)

		childBlocks := make([][]string, 0, len(child))
		for _, cutAt := range overridesFromChild {
			childBlocks = append(childBlocks, child[:cutAt+1])
			if cutAt == len(child) {
				child = []string{}
			} else {
				child = child[cutAt+1:]
			}
		}
		childBlocks = append(childBlocks, child)

		if config.Trace {
			log.Printf("Lifecycle order overrides:\n\tparent: %v\n\tchild: %v", parentBlocks, childBlocks)
		}

		for i, parentBlock := range parentBlocks {
			order = append(order, parentBlock...)
			order = append(order, childBlocks[i]...)
		}
	}
	return order
}

func mergeReadyCondition(parent, child []manifest.ReadyCondition) []manifest.ReadyCondition {
	cond := make([]manifest.ReadyCondition, 0, len(parent)+len(child))
	cond = append(cond, parent...)
	cond = append(cond, child...)
	return cond
}

func mergeRequiresTuning(parent, child manifest.RequiresTuning) manifest.RequiresTuning {
	newestFirst := util.MergeUnique(util.Reverse(child.Optional), util.Reverse(parent.Optional))
	merged := make([]string, 0, len(newestFirst))

	eraseRequirement := make([]string, 0)
	eraseComponent := make([]string, 0)

	// go back to front skipping entries that are overriden by newer entries
	for _, req := range newestFirst {
		skip := false
		i := strings.Index(req, ":")

		if i == -1 { // plain requirement ie. `vault` which is effectively a `vault:*`

			// we have seen a newer `requirement:` spec which means - forget about `requirement` tuning
			if util.Contains(eraseRequirement, req) {
				continue
			}

			for _, seen := range merged {
				// skip if a fine-grained requirement is defined
				if strings.HasPrefix(seen, req+":") {
					skip = true
					break
				}
			}
		} else {
			if req == ":" { // erase everything that is older
				break
			}

			if i > 0 {
				if i < len(req)-1 { // requirement:component
					component := req[i+1:]
					plainReq := req[:i]
					if util.Contains(eraseComponent, component) || // seen `:component`
						util.Contains(eraseRequirement, plainReq) ||
						util.Contains(merged, plainReq) { // a req:* is specified

						skip = true
					}
				} else { // requirement:
					// skip all older specs for `requirement`
					eraseRequirement = append(eraseRequirement, req[:i])
					skip = true
				}
			} else { // :component
				// skip all older specs for `component`
				eraseComponent = append(eraseComponent, req[i:])
				skip = true
			}
		}

		if !skip {
			merged = append(merged, req)
		}
	}
	return manifest.RequiresTuning{util.Reverse(merged)}
}

func mergeOutputs(parent, child []manifest.Output) []manifest.Output {
	outputs := make([]manifest.Output, 0, len(parent)+len(child))
	outputs = append(outputs, parent...)
	for _, output := range child {
		found := false
		for i, exist := range outputs {
			if output.Name == exist.Name {
				found = true
				outputs[i] = manifest.Output{
					Name:        output.Name,
					Value:       mergeField(exist.Value, output.Value),
					FromTfVar:   mergeField(exist.FromTfVar, output.FromTfVar),
					Kind:        mergeField(exist.Kind, output.Kind),
					Brief:       mergeField(exist.Brief, output.Brief),
					Description: mergeField(exist.Description, output.Description),
				}
				break
			}
		}
		if !found {
			outputs = append(outputs, output)
		}
	}
	return outputs
}

func mergeComponentsRefs(parentBaseDir, componentsBaseDir string,
	parent, child []manifest.ComponentRef) []manifest.ComponentRef {

	refs := make([]manifest.ComponentRef, 0, len(parent)+len(child))
	for _, ref := range parent {
		if ref.Source.Dir != "" {
			ref.Source.Dir = filepath.Join(parentBaseDir, ref.Source.Dir)
		}
		if ref.Source.Git.LocalDir != "" {
			if !filepath.IsAbs(ref.Source.Git.LocalDir) {
				ref.Source.Git.LocalDir = filepath.Join(componentsBaseDir, ref.Source.Git.LocalDir)
			}
		} else if ref.Source.Git.Remote != "" && parentBaseDir == componentsBaseDir {
			ref.Source.Git.LocalDir = filepath.Join(parentBaseDir, manifest.ComponentSourceDirNameFromRef(&ref))
		}
		refs = append(refs, ref)
	}
	for _, ref := range child {
		found := false
		for i, exist := range refs {
			if ref.Name == exist.Name {
				found = true
				refs[i] = ref
				break
			}
		}
		if !found {
			refs = append(refs, ref)
		}
	}
	return refs
}

func mergeComponentsManifests(parent, child []manifest.Manifest) []manifest.Manifest {
	manifests := make([]manifest.Manifest, 0, len(parent)+len(child))
	manifests = append(manifests, parent...)
	for _, manifest := range child {
		found := false
		for i, exist := range manifests {
			if manifest.Meta.Name == exist.Meta.Name {
				found = true
				manifests[i] = manifest
				break
			}
		}
		if !found {
			manifests = append(manifests, manifest)
		}
	}
	return manifests

}

func writeStackManifest(elaborateManifests []string, stackManifest *manifest.Manifest, componentsManifest []manifest.Manifest) error {
	elaborateFiles, errs := storage.Check(elaborateManifests, "elaborate")
	if len(errs) > 0 {
		log.Fatalf("Unable to check elaborate files: %v", util.Errors2(errs...))
	}

	var yamlBytes bytes.Buffer

	yamlDocSeparator := []byte("---\n")
	stackManifest.Document = ""
	marshaled, err := yaml.Marshal(stackManifest)
	if err != nil {
		return err
	}
	yamlBytes.Write(yamlDocSeparator)
	written, err := yamlBytes.Write(marshaled)
	if err != nil || written != len(marshaled) {
		return fmt.Errorf("Buffer write failed %v; wrote %d out of %d bytes", err, len(marshaled), written)
	}
	for _, componentManifest := range componentsManifest {
		componentManifest.Document = ""
		marshaled, err := yaml.Marshal(componentManifest)
		if err != nil {
			return err
		}
		yamlBytes.Write(yamlDocSeparator)
		written, err := yamlBytes.Write(marshaled)
		if err != nil || written != len(marshaled) {
			return fmt.Errorf("Buffer write failed %v; wrote %d out of %d bytes", err, len(marshaled), written)
		}
	}

	errs = storage.Write(yamlBytes.Bytes(), elaborateFiles)
	if len(errs) > 0 {
		log.Fatalf("Unable to write elaborate: %s", util.Errors2(errs...))
	}

	return nil
}
