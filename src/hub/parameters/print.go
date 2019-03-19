package parameters

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"hub/util"
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
		log.Printf("\t%s => `%s`%s", parameter.QName(), util.Wrap(parameter.Value), env)
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
		kind := ""
		if output.Kind != "" {
			kind = fmt.Sprintf("[%s] ", output.Kind)
		}
		log.Printf("\t%s%s:%s => `%s`", kind, output.Component, output.Name, util.Wrap(output.Value))
	}
}

func PrintCapturedOutputsByComponent(outputs CapturedOutputs, component string) {
	PrintCapturedOutputsList(CapturedOutputsToListByComponent(outputs, component))
}

func PrintCapturedOutputs(outputs CapturedOutputs) {
	PrintCapturedOutputsList(CapturedOutputsToList(outputs))
}
