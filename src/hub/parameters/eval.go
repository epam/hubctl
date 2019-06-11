package parameters

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"hub/config"
	"hub/manifest"
	"hub/util"
)

var CurlyReplacement = regexp.MustCompile("[\\$#][^}]+}")

func StripCurly(match string) (string, bool) {
	return match[2 : len(match)-1], match[0] == '#'
}

func RequireExpansion(value string) bool {
	return strings.Contains(value, "${") || strings.Contains(value, "#{")
}

func LockParameters(parameters []manifest.Parameter,
	extraValues []manifest.Parameter,
	ask func(*manifest.Parameter) error) (LockedParameters, []error) {

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
			err := ask(parameter)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	// create key-value map for parameter expansion
	kv := make(map[string]string)
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

func ExpandParameter(parameter *manifest.Parameter, componentDepends []string, kv map[string]string) []error {
	value, errs := expandValue(parameter, parameter.Value, componentDepends, kv, 0)
	parameter.Value = value
	return errs
}

const maxExpansionDepth = 10

func expandValue(parameter *manifest.Parameter, value string, componentDepends []string,
	kv map[string]string, depth int) (string, []error) {

	fqName := parameter.QName()
	if depth >= maxExpansionDepth {
		return "(loop)", []error{fmt.Errorf("Probably loop expanding parameter `%s` value `%s`, reached `%s` at depth %d",
			fqName, parameter.Value, value, depth)}
	}
	errs := make([]error, 0)
	expandedValue := CurlyReplacement.ReplaceAllStringFunc(value,
		func(match string) string {
			expr, isCel := StripCurly(match)
			// TODO review: expansion search path is set to parameter's `component:`
			// but in hub-component.yaml it may lead to the path being set to an
			// unexpected qualifier or not being set at all
			var substitution string
			if isCel {
				var err error
				substitution, err = CelEval(expr, parameter.Component, componentDepends, kv)
				if err != nil {
					errs = append(errs, err)
				}
			} else {
				var exist bool
				substitution, exist = FindValue(expr, parameter.Component, componentDepends, kv)
				if !exist {
					errs = append(errs, fmt.Errorf("Parameter `%s` value `%s` refer to unknown substitution `%s` at depth %d",
						parameter.QName(), parameter.Value, expr, depth))
					substitution = "(unknown)"
				}
			}
			if config.Trace {
				log.Printf("--- %s | %s => %s", expr, parameter.Component, substitution)
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

		if config.Trace {
			log.Printf("--- %s | %s => %s", parameter.Name, componentName, parameter.Value)
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
	fqName := lockedParameterQualifiedName(&add)
	current, exists := parameters[fqName]
	if exists {
		if current.Value != add.Value && current.Value != "" {
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
