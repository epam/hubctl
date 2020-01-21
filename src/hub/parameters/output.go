package parameters

import (
	"fmt"
	"log"
	"strings"

	"hub/config"
	"hub/manifest"
	"hub/util"
)

func MergeOutputs(outputs CapturedOutputs, toMerge CapturedOutputs) {
	for _, o := range toMerge {
		MergeOutput(outputs, o)
	}
}

func OutputsFromList(toMerge ...[]CapturedOutput) CapturedOutputs {
	outputs := make(CapturedOutputs)
	for _, list := range toMerge {
		for _, o := range list {
			MergeOutput(outputs, o)
		}
	}
	return outputs
}

func MergeOutput(outputs CapturedOutputs, add CapturedOutput) {
	qName := add.QName()
	if config.Verbose {
		if current, exists := outputs[qName]; exists && !util.Empty(current.Value) {
			curValue := util.String(current.Value)
			addValue := util.String(add.Value)
			if curValue != addValue &&
				// suppress warning if plain value overwritten by secret
				!(current.Kind == "" && strings.HasPrefix(add.Kind, "secret")) {

				log.Printf("Output `%s` current value `%s` overridden by new value `%s`",
					qName, util.Wrap(curValue), util.Wrap(addValue))
			}
		}
	}
	outputs[qName] = add
}

func ExpandRequestedOutputs(parameters LockedParameters, outputs CapturedOutputs,
	requestedOutputs []manifest.Output, mustExist bool) []ExpandedOutput {

	kvParameters := ParametersKV(parameters)
	kvOutputs := OutputsKV(outputs)

	expanded := make([]ExpandedOutput, 0, len(outputs))
	debugPrinted := false

	for _, requestedOutput := range requestedOutputs {
		var value interface{}
		kind := requestedOutput.Kind
		brief := requestedOutput.Brief
		valueExist := false
		// output from a specific component
		componentOutputRequested := strings.Contains(requestedOutput.Name, ":")

		if !componentOutputRequested && util.Empty(requestedOutput.Value) && requestedOutput.Name != "" {
			requestedOutput.Value = fmt.Sprintf("${%s}", requestedOutput.Name)
		}

		if componentOutputRequested {
			if !util.Empty(requestedOutput.Value) {
				util.Warn("Stack output `%s` refer to value `%s`, but it will be derived from component outputs",
					requestedOutput.Name, requestedOutput.Value)
			}
			var exist bool
			value, exist = kvOutputs[requestedOutput.Name]
			if !exist && mustExist {
				util.Warn("Stack output `%s` not found in outputs:", requestedOutput.Name)
				if !debugPrinted {
					PrintCapturedOutputs(outputs)
					debugPrinted = true
				}
			}
			// propagate output (secret) kind and brief
			if exist && (kind == "" || brief == "") {
				for _, o := range outputs {
					if o.QName() == requestedOutput.Name {
						if o.Kind != "" && kind == "" {
							kind = o.Kind
						}
						if o.Brief != "" && brief == "" {
							brief = o.Brief
						}
						break
					}
				}
			}
			valueExist = exist
		} else if RequireExpansion(requestedOutput.Value) {
			requestedOutputValue := requestedOutput.Value.(string)
			invoked := 0
			found := 0
			value = CurlyReplacement.ReplaceAllStringFunc(requestedOutputValue,
				func(match string) string {
					invoked++
					variable, isCel := StripCurly(match)
					if isCel {
						util.Warn("Stack output `%s = %s` CEL expression `%s` is not supported",
							requestedOutput.Name, requestedOutputValue, variable)
						return "(unsupported)"
					}
					substitution, exist := kvOutputs[variable]
					if !exist {
						substitution, exist = kvParameters[variable]
					}
					if !exist {
						if mustExist {
							util.Warn("Stack output `%s = %s` refer to unknown substitution `%s`",
								requestedOutput.Name, requestedOutputValue, variable)
						}
					} else if RequireExpansion(substitution) {
						log.Fatalf("Stack output `%s = %s` refer to substitution `%s` that expands to `%s`. This is surely a bug.",
							requestedOutput.Name, requestedOutputValue, variable, substitution)
					} else {
						found++
					}
					if str, ok := substitution.(string); ok {
						return str
					} else {
						return fmt.Sprintf("%v", substitution)
					}
				})
			valueExist = invoked == found
		} else {
			value = requestedOutput.Value
			valueExist = true
		}

		if valueExist {
			expanded = append(expanded,
				ExpandedOutput{Name: requestedOutput.Name, Value: value, Kind: kind, Brief: brief})
		}
	}

	return expanded
}
