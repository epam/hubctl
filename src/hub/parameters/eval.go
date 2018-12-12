package parameters

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/mattn/go-isatty"

	"hub/api"
	"hub/config"
	"hub/manifest"
	"hub/util"
)

var CurlyReplacement = regexp.MustCompile("\\${[a-zA-Z0-9_\\.\\|:/-]+?}")
var mustacheReplacement = regexp.MustCompile("{{[a-zA-Z0-9_\\.\\|:/-]+?}}")

func StripCurly(match string) string {
	return match[2 : len(match)-1]
}

func LockParameters(parameters []manifest.Parameter,
	environment map[string]string, extraValues []manifest.Parameter,
	hubEnvironment, hubStackInstance, hubApplication string,
	isDeploy bool) LockedParameters {

	for _, parameter := range parameters {
		if parameter.Default != "" && parameter.Kind != "user" {
			kind := ""
			if parameter.Kind != "" {
				kind = fmt.Sprintf(" but `%s`", parameter.Kind)
			}
			util.Warn("Parameter `%s` default value `%s` won't be used as parameter is not `user` kind%s",
				parameter.Name, util.Wrap(parameter.Default), kind)
		}

	}
	errs := make([]error, 0)
	// populate empty user-level parameters from environment or user input
	for i, _ := range parameters {
		parameter := &parameters[i]
		if parameter.Value == "" && parameter.Kind == "user" && len(parameter.Parameters) == 0 {
			err := AskParameter(parameter, environment, hubEnvironment, hubStackInstance, hubApplication, isDeploy)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	// create key-value map for parameter expansion
	kv := make(map[string]string)
	for _, extra := range extraValues {
		fqName := manifestParameterQualifiedName(&extra)
		kv[fqName] = extra.Value
	}
	for _, parameter := range parameters {
		fqName := manifestParameterQualifiedName(&parameter)
		kv[fqName] = parameter.Value
	}
	// expand, check for cycles
	locked := make(LockedParameters)
	for _, parameter := range parameters {
		fqName := manifestParameterQualifiedName(&parameter)
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
	if len(errs) > 0 {
		log.Fatalf("Failed to lock stack parameters:\n\t%s", util.Errors("\n\t", errs...))
	}
	return locked
}

func AskParameter(parameter *manifest.Parameter,
	environment map[string]string, hubEnvironment, hubStackInstance, hubApplication string,
	isDeploy bool) (retErr error) {

	if parameter.FromEnv != "" {
		key := parameter.FromEnv
		if environment != nil {
			if v, exist := environment[key]; exist {
				parameter.Value = v
				return
			}
		}
		if v, exist := os.LookupEnv(key); exist {
			parameter.Value = v
			return
		}
	}

	qName := manifestParameterQualifiedName(parameter)

	if hubEnvironment != "" || hubStackInstance != "" || hubApplication != "" {
		found, v, err := api.GetParameterOrMaybeCreatePassword(hubEnvironment, hubStackInstance, hubApplication,
			parameter.Name, parameter.Component, isDeploy)
		if err != nil {
			where := make([]string, 0, 3)
			if hubEnvironment != "" {
				where = append(where, fmt.Sprintf("environment `%s`", hubEnvironment))
			}
			if hubStackInstance != "" {
				where = append(where, fmt.Sprintf("stack instance `%s`", hubStackInstance))
			}
			if hubApplication != "" {
				where = append(where, fmt.Sprintf("application `%s`", hubApplication))
			}
			util.Warn("Error query parameter `%s` in %s: %v",
				qName, strings.Join(where, ", "), err)
		} else if found && v != "" {
			parameter.Value = v
			return
		}
	}

	// TODO review
	// if parameter with default value is marked empty: allow, then we set value to default without prompt
	if parameter.Empty != "allow" && isatty.IsTerminal(os.Stdin.Fd()) {
		prompt := "Enter value for"
		if parameter.Brief != "" {
			prompt = fmt.Sprintf("%s %s (%s)", prompt, parameter.Brief, qName)
		} else {
			prompt = fmt.Sprintf("%s %s", prompt, qName)
		}
		if parameter.Default != "" {
			prompt = fmt.Sprintf("%s [%s]", prompt, parameter.Default)
		}
		fmt.Printf("%s: ", prompt)
		read, err := fmt.Scanln(&parameter.Value)
		if read > 0 {
			if err != nil {
				log.Fatalf("Error reading input: %v (read %d items)", err, read)
			} else {
				return
			}
		}
	}

	if parameter.Default == "" {
		if parameter.Env != "" && parameter.FromEnv == "" {
			util.Warn("Parameter `%s` has `env = %s` assigned. Did you mean `fromEnv`?", qName, parameter.Env)
		}
		if parameter.Empty == "allow" {
			if config.Debug {
				log.Printf("Empty parameter `%s` value allowed", qName)
			}
			return
		}
		retErr = fmt.Errorf("Parameter `%s` has no value nor default assigned", qName)
		parameter.Value = "unknown"
	} else {
		parameter.Value = parameter.Default
	}

	return
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

func RequireExpansion(value string) bool {
	return strings.Contains(value, "${")
}

func ExpandParameter(parameter *manifest.Parameter, componentDepends []string, kv map[string]string) []error {
	value, errs := expandValue(parameter, parameter.Value, componentDepends, kv, 0)
	parameter.Value = value
	return errs
}

const maxExpansionDepth = 10

func expandValue(parameter *manifest.Parameter, value string, componentDepends []string,
	kv map[string]string, depth int) (string, []error) {

	fqName := manifestParameterQualifiedName(parameter)
	if depth >= maxExpansionDepth {
		return "(loop)", []error{fmt.Errorf("Probably loop expanding parameter `%s` value `%s`, reached `%s` at depth %d",
			fqName, parameter.Value, value, depth)}
	}
	errs := make([]error, 0)
	expandedValue := CurlyReplacement.ReplaceAllStringFunc(value,
		func(variable string) string {
			variable = StripCurly(variable)
			substitution, exist := FindValue(variable, parameter.Component, componentDepends, kv)
			if !exist {
				errs = append(errs, fmt.Errorf("Parameter `%s` value `%s` refer to unknown substitution `%s` at depth %d",
					manifestParameterQualifiedName(parameter), parameter.Value, variable, depth))
				substitution = "(unknown)"
			}
			if config.Trace {
				log.Printf("--- %s | %s => %s", variable, parameter.Component, substitution)
			}
			if RequireExpansion(substitution) {
				var nestedErrs []error
				substitution, nestedErrs = expandValue(parameter, substitution, componentDepends, kv, depth+1)
				errs = append(errs, nestedErrs...)
			}
			return substitution
		})
	if depth == 0 && config.Debug { // do not change to Trace
		log.Printf("--- %s `%s` => `%s`", fqName, strings.TrimSpace(value), strings.TrimSpace(expandedValue))
	}
	return expandedValue, errs
}

func ParametersKV(parameters LockedParameters) map[string]string {
	kv := make(map[string]string)
	for _, parameter := range parameters {
		fqName := lockedParameterQualifiedName(&parameter)
		kv[fqName] = parameter.Value
	}
	return kv
}

func outputsKV(outputs CapturedOutputs) map[string]string {
	kv := make(map[string]string)
	for _, output := range outputs {
		kv[output.QName()] = output.Value
		kv[output.Name] = output.Value
	}
	return kv
}

func ParametersAndOutputsKV(parameters LockedParameters, outputs CapturedOutputs) map[string]string {
	kv := make(map[string]string)
	for _, parameter := range parameters {
		kv[parameter.QName()] = parameter.Value
	}
	for _, output := range outputs {
		kv[output.QName()] = output.Value
		kv[output.Name] = output.Value
	}
	return kv
}

func FindValue(parameterName string, componentName string, componentDepends []string,
	kv map[string]string) (string, bool) {

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
		for _, component := range componentDepends {
			relatedParameterName := parameterQualifiedName(parameterName, component)
			v, exist = kv[relatedParameterName]
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

func ExpandParameters(componentName string, componentDepends []string,
	parameters LockedParameters, outputs CapturedOutputs,
	componentParameters []manifest.Parameter, environment map[string]string) ([]LockedParameter, []error) {

	kv := ParametersAndOutputsKV(parameters, outputs)
	kv["component.name"] = componentName // TODO deprecate
	kv["hub.componentName"] = componentName
	// expand, check for cycles
	expanded := make([]LockedParameter, 0, len(componentParameters)+3)
	expanded = append(expanded, LockedParameter{Name: "component.name", Value: componentName})
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
				AskParameter(&parameter, environment, "", "", "", false)
				if RequireExpansion(parameter.Value) {
					errs = append(errs, ExpandParameter(&parameter, componentDepends, kv)...)
				}
			} else {
				if parameter.Value == "" && parameter.Default != "" {
					parameter.Value = parameter.Default
				}
				if parameter.Value == "" {
					if parameter.Empty == "allow" {
						if config.Debug {
							log.Printf("Empty parameter `%s` value allowed", fqName)
						}
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
		}

		if config.Trace {
			log.Printf("--- %s | %s => %s", parameter.Name, componentName, parameter.Value)
		}

		expanded = append(expanded, LockedParameter{Name: parameter.Name, Value: parameter.Value, Env: parameter.Env})
	}
	if config.Debug && len(expanded) > 1 {
		log.Print("Parameters expanded:")
		PrintLockedParametersList(expanded)
	}
	if len(errs) > 0 && !config.Force {
		if !config.Debug {
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
	fqName := lockedParameterQualifiedName(&add)
	current, exists := parameters[fqName]
	if exists {
		if current.Value != add.Value && current.Value != "" && fqName != "component.name" {
			util.Warn("Parameter `%s` current value `%s` does not match new value `%s`",
				fqName, current.Value, add.Value)
		}
		if current.Env != "" && add.Env != "" && current.Env != add.Env {
			util.Warn("Parameter `%s` environment variable setup `%s` does not match new setup `%s`",
				fqName, current.Env, add.Env)
		}

	}
	parameters[fqName] = add
}

func MergeParameters(parameters LockedParameters, componentParameters ...[]LockedParameter) LockedParameters {
	merged := make(LockedParameters)
	for k, v := range parameters {
		merged[k] = v
	}
	for _, list := range componentParameters {
		for _, p := range list {
			mergeParameter(merged, p)
		}
	}
	return merged
}
