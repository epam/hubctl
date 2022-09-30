// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parameters

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

func LockedParametersToList(parameters LockedParameters) []LockedParameter {
	keys := make([]string, 0, len(parameters))
	for name := range parameters {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	list := make([]LockedParameter, 0, len(parameters))
	for _, name := range keys {
		list = append(list, parameters[name])
	}
	return list
}

func PrintLockedParametersList(parameters []LockedParameter) {
	for _, parameter := range parameters {
		env := ""
		if parameter.Env != "" {
			env = fmt.Sprintf(" (env:%s)", parameter.Env)
		}
		value := util.String(parameter.Value)
		if !config.Trace && util.LooksLikeSecret(parameter.Name) && len(value) > 0 {
			value = "(masked)"
		} else {
			value = fmt.Sprintf("`%s`", util.Wrap(value))
		}
		log.Printf("\t%s => %s%s", parameter.QName(), value, env)
	}
}

func PrintLockedParameters(parameters LockedParameters) {
	PrintLockedParametersList(LockedParametersToList(parameters))
}

func RawOutputsToList(outputs RawOutputs) []RawOutput {
	if len(outputs) == 0 {
		return []RawOutput{}
	}
	keys := make([]string, 0, len(outputs))
	for name := range outputs {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	list := make([]RawOutput, 0, len(outputs))
	for _, name := range keys {
		list = append(list, RawOutput{Name: name, Value: outputs[name]})
	}
	return list
}

func CapturedOutputsToListByComponent(outputs CapturedOutputs, component string) []CapturedOutput {
	if component != "" {
		component = component + ":"
	}

	keys := make([]string, 0, len(outputs))
	for name := range outputs {
		if component == "" || strings.HasPrefix(name, component) {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)

	list := make([]CapturedOutput, 0, len(outputs))
	for _, name := range keys {
		list = append(list, outputs[name])
	}
	return list
}

func CapturedOutputsToList(outputs CapturedOutputs) []CapturedOutput {
	return CapturedOutputsToListByComponent(outputs, "")
}

func PrintCapturedOutputsList(outputs []CapturedOutput) {
	for _, output := range outputs {
		value := fmt.Sprintf("`%s`", util.Wrap(util.String(output.Value)))
		kind := ""
		if output.Kind != "" {
			kind = fmt.Sprintf("[%s] ", output.Kind)
			if !config.Trace && strings.HasPrefix(output.Kind, "secret") && !util.Empty(output.Value) {
				value = "(masked)"
			}
		}
		brief := ""
		if output.Brief != "" {
			brief = fmt.Sprintf(" [%s]", output.Brief)
		}
		log.Printf("\t%s%s:%s%s => %s", kind, output.Component, output.Name, brief, value)
	}
}

func PrintCapturedOutputsByComponent(outputs CapturedOutputs, component string) {
	PrintCapturedOutputsList(CapturedOutputsToListByComponent(outputs, component))
}

func PrintCapturedOutputs(outputs CapturedOutputs) {
	PrintCapturedOutputsList(CapturedOutputsToList(outputs))
}
