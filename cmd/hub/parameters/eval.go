package parameters

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/util"
)

// ${var.name} or
// #{cel - expression {optionaly one nested level of braces for maps}}
var CurlyReplacement = regexp.MustCompile("\\$\\{[^}]+\\}|#\\{(?:[^}{]|\\{[^}{]+\\})*\\}")

func StripCurly(match string) (string, bool) {
	return match[2 : len(match)-1], match[0] == '#'
}

func RequireExpansion(value interface{}) bool {
	if str, ok := value.(string); ok {
		return strings.Contains(str, "${") || strings.Contains(str, "#{")
	}
	return false
}

func LockParameters(parameters []manifest.Parameter,
	extraValues []manifest.Parameter,
	ask func(manifest.Parameter) (interface{}, error)) (LockedParameters, []error) {

	for _, parameter := range parameters {
		if !util.Empty(parameter.Default) && parameter.Kind != "user" {
			kind := ""
			if parameter.Kind != "" {
				kind = fmt.Sprintf(" but `%s`", parameter.Kind)
			}
			util.Warn("Parameter `%s` default value `%s` won't be used as parameter is not `user` kind%s",
				parameter.Name, util.Wrap(util.String(parameter.Default)), kind)
		}
	}
	errs := make([]error, 0)
	// populate empty user-level parameters from environment or user input
	for i, parameter := range parameters {
		if util.Empty(parameter.Value) && parameter.Kind == "user" && len(parameter.Parameters) == 0 {
			value, err := ask(parameter)
			parameters[i].Value = value
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	// create key-value map for parameter expansion
	kv := make(map[string]interface{})
	for _, extra := range extraValues {
		kv[extra.QName()] = extra.Value
	}
	for _, parameter := range parameters {
		kv[parameter.QName()] = parameter.Value
	}
	// expand, check for cycles
	locked := make(LockedParameters)
	for _, parameter := range parameters {
		fqName := parameter.QName()
		if RequireExpansion(parameter.Value) && parameter.Kind != "link" {
			errs = append(errs, ExpandParameter(&parameter, []string{}, kv)...)
			kv[fqName] = parameter.Value
		}
		locked[fqName] = LockedParameter{Name: parameter.Name, Component: parameter.Component,
			Value: parameter.Value, Env: parameter.Env}
	}
	if config.Debug && len(locked) > 0 {
		log.Print("Parameters locked:")
		PrintLockedParameters(locked)
	}
	return locked, errs
}

func ParametersWithoutLinks(parameters LockedParameters) LockedParameters {
	noLinks := true
	for _, parameter := range parameters {
		if RequireExpansion(parameter.Value) {
			noLinks = false
			break
		}
	}
	if noLinks {
		return parameters
	}
	withoutLinks := make(LockedParameters)
	for key, parameter := range parameters {
		if !RequireExpansion(parameter.Value) {
			withoutLinks[key] = parameter
		}
	}
	return withoutLinks
}

func ExpandParameter(parameter *manifest.Parameter, componentDepends []string, kv map[string]interface{}) []error {
	str, ok := parameter.Value.(string)
	if !ok {
		return []error{fmt.Errorf("Unable to expand parameter `%s` value `%+v`, which is not a string",
			parameter.QName(), parameter.Value)}
	}
	value, errs, _ := expandValue(parameter, str, componentDepends, kv, 0)
	parameter.Value = value
	return errs
}

const maxExpansionDepth = 10

func expandValue(parameter *manifest.Parameter, value string, componentDepends []string,
	kv map[string]interface{}, depth int) (string, []error, bool) {

	if depth >= maxExpansionDepth {
		return "(loop)", []error{fmt.Errorf("Probably loop expanding parameter `%s` value `%s`, reached `%s` at depth %d",
			parameter.QName(), parameter.Value, value, depth)}, false
	}
	errs := make([]error, 0)
	mask := util.LooksLikeSecret(parameter.Name)
	expandedValue := CurlyReplacement.ReplaceAllStringFunc(value,
		func(match string) string {
			expr, isCel := StripCurly(match)
			// TODO review: expansion search path is set to parameter's `component:`
			// but in hub-component.yaml it may lead to the component being set to an
			// unexpected qualifier or not being set at all
			var substitution string
			if isCel {
				evaluated, err := CelEval(expr, parameter.Component, componentDepends, kv)
				if err != nil {
					errs = append(errs, err)
				}
				substitution = evaluated
			} else {
				mask = mask || util.LooksLikeSecret(expr)
				found, exist := FindValue(expr, parameter.Component, componentDepends, kv)
				if !exist {
					errs = append(errs, fmt.Errorf("Parameter `%s` value `%s` refer to unknown substitution `%s` at depth %d",
						parameter.QName(), parameter.Value, expr, depth))
					substitution = "(unknown)"
				} else {
					if found == nil {
						substitution = ""
					} else if str, ok := found.(string); ok {
						substitution = str
					} else {
						substitution = fmt.Sprintf("%v", found)
					}
				}
			}
			if config.Trace {
				log.Printf("--- %s => %s", manifest.ParameterQualifiedName(expr, parameter.Component), substitution)
			}
			if RequireExpansion(substitution) {
				expanded, errs2, mask2 := expandValue(parameter, substitution, componentDepends, kv, depth+1)
				mask = mask || mask2
				errs = append(errs, errs2...)
				substitution = expanded
			}
			return substitution
		})
	if depth == 0 && config.Debug { // do not change to Trace
		print := fmt.Sprintf("`%s`", expandedValue)
		if !config.Trace && mask && expandedValue != "" {
			print = "(masked)"
		}
		log.Printf("--- %s `%s` => %s", parameter.QName(), value, print)
	}
	return expandedValue, errs, mask
}

func ParametersKV(parameters LockedParameters) map[string]interface{} {
	kv := make(map[string]interface{})
	for _, parameter := range parameters {
		kv[parameter.QName()] = parameter.Value
	}
	return kv
}

func OutputsKV(outputs CapturedOutputs) map[string]interface{} {
	kv := make(map[string]interface{})
	for _, output := range outputs {
		kv[output.QName()] = output.Value
		kv[output.Name] = output.Value
	}
	return kv
}

func ParametersAndOutputsKV(parameters LockedParameters, outputs CapturedOutputs, outputFilter func(CapturedOutput) bool) map[string]interface{} {
	kv := make(map[string]interface{})
	for _, parameter := range parameters {
		kv[parameter.QName()] = parameter.Value
	}
	for _, output := range outputs {
		if outputFilter != nil && !outputFilter(output) {
			if config.Trace {
				log.Printf("Output `%s` hidden", output.QName())
			}
			continue
		}
		kv[output.QName()] = output.Value
		kv[output.Name] = output.Value
	}
	return kv
}

func ParametersAndOutputsKVWithDepends(parameters LockedParameters, outputs CapturedOutputs, depends []string) map[string]interface{} {
	kv := make(map[string]interface{})
	for _, parameter := range parameters {
		kv[parameter.QName()] = parameter.Value
	}
	for _, output := range outputs {
		kv[output.QName()] = output.Value
		kv[output.Name] = output.Value
	}
	// for Go Template and Mustache bindings we rearrange outputs to have a plain non-Qname
	// to take precedence according to `depends`
	for _, dependsOn := range util.Reverse(depends) {
		for _, output := range outputs {
			if output.Component == dependsOn {
				kv[output.Name] = output.Value
			}
		}
	}
	return kv
}

func FindValue(parameterName string, componentName string, componentDepends []string,
	kv map[string]interface{}) (interface{}, bool) {

	fqName := parameterQualifiedName(parameterName, componentName)
	v, exist := kv[fqName]
	if !exist {
		for _, outputComponent := range componentDepends {
			outputFqName := OutputQualifiedName(parameterName, outputComponent)
			v, exist = kv[outputFqName]
			if exist {
				break
			}
		}
	}
	if !exist {
		v, exist = kv[parameterName]
	}
	return v, exist
}

func ExpandParameters(componentName, componentKind string, componentDepends []string,
	parameters LockedParameters, outputs CapturedOutputs,
	componentParameters []manifest.Parameter) ([]LockedParameter, []error) {

	// make outputs of the same component kind invisible
	outputFilter := func(output CapturedOutput) bool {
		return !(componentKind != "" && componentKind == output.ComponentKind)
	}
	kv := ParametersAndOutputsKV(parameters, outputs, outputFilter)
	kv["hub.componentName"] = componentName
	// expand, check for cycles
	expanded := make([]LockedParameter, 0, len(componentParameters)+3)
	expanded = append(expanded, LockedParameter{Name: "hub.componentName", Value: componentName})
	errs := make([]error, 0)
	for _, parameter := range componentParameters {
		fqName := parameterQualifiedName(parameter.Name, componentName)
		v, exist := FindValue(parameter.Name, componentName, componentDepends, kv)
		if exist {
			parameter.Value = v
			if RequireExpansion(parameter.Value) {
				errs = append(errs, ExpandParameter(&parameter, componentDepends, kv)...)
			}
		} else {
			if parameter.Kind == "user" {
				util.Warn("Component `%s` user-level parameter `%s` must be propagated to stack level parameter",
					componentName, fqName)
			}
			if util.Empty(parameter.Value) && !util.Empty(parameter.Default) {
				parameter.Value = parameter.Default
			}
			if util.Empty(parameter.Value) {
				if parameter.Empty == "allow" {
					if config.Debug {
						log.Printf("Empty parameter `%s` value allowed", fqName)
					}
					parameter.Value = ""
				} else {
					errs = append(errs, fmt.Errorf("Parameter `%s` value cannot be derived from stack parameters nor outputs", fqName))
					parameter.Value = "(unknown)"
				}
			} else {
				if RequireExpansion(parameter.Value) {
					errs = append(errs, ExpandParameter(&parameter, componentDepends, kv)...)
				}
			}
		}

		if config.Trace {
			log.Printf("--- %s | %s => %v", parameter.Name, componentName, parameter.Value)
		}

		expanded = append(expanded, LockedParameter{Name: parameter.Name, Value: parameter.Value, Env: parameter.Env})
		kv[parameter.Name] = parameter.Value
	}
	if config.Trace && len(expanded) > 1 {
		log.Print("Parameters expanded:")
		PrintLockedParametersList(expanded)
	}
	if len(errs) > 0 && !config.Force {
		if !config.Trace {
			log.Print("Parameters expanded:")
			PrintLockedParametersList(expanded)
		}
		log.Print("Currently known stack parameters:")
		PrintLockedParameters(parameters)
		if len(outputs) > 0 {
			log.Print("Currently known outputs:")
			PrintCapturedOutputs(outputs)
		}
	}

	return expanded, errs
}

func mergeParameter(parameters LockedParameters, add LockedParameter) {
	qName := add.QName()
	current, exists := parameters[qName]
	if exists {
		curValue := util.String(current.Value)
		addValue := util.String(add.Value)
		if curValue != addValue && !util.Empty(current.Value) {
			util.Warn("Parameter `%s` current value `%s` does not match new value `%s`",
				qName,
				util.Trim(util.MaybeMaskedValue(config.Trace, qName, curValue)),
				util.Trim(util.MaybeMaskedValue(config.Trace, qName, addValue)))
		}
		if current.Env != "" && add.Env != "" && current.Env != add.Env {
			util.Warn("Parameter `%s` environment variable setup `%s` does not match new setup `%s`",
				qName, current.Env, add.Env)
		}

	}
	parameters[qName] = add
}

func MergeParameters(parameters LockedParameters, toMerge ...[]LockedParameter) LockedParameters {
	merged := make(LockedParameters)
	if parameters != nil {
		for k, v := range parameters {
			merged[k] = v
		}
	}
	for _, list := range toMerge {
		for _, p := range list {
			mergeParameter(merged, p)
		}
	}
	return merged
}

func ParametersFromList(toMerge ...[]LockedParameter) LockedParameters {
	merged := make(LockedParameters)
	for _, list := range toMerge {
		for _, p := range list {
			mergeParameter(merged, p)
		}
	}
	return merged
}
