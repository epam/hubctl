package lifecycle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

func captureOutputs(componentName string, componentParameters parameters.LockedParameters,
	textOutput []byte, requestedOutputs []manifest.Output) (parameters.RawOutputs, parameters.CapturedOutputs, []string, []error) {

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
					func(match string) string {
						expr, isCel := parameters.StripCurly(match)
						var substitution string
						if isCel {
							var err error
							substitution, err = parameters.CelEval(expr, componentName, nil, kv)
							if err != nil {
								errs = append(errs, err)
							}
						} else {
							var exist bool
							substitution, exist = parameters.FindValue(expr, componentName, nil, kv)
							if !exist {
								errs = append(errs, fmt.Errorf("Component `%s` output `%s = %s` refer to unknown substitution `%s`",
									componentName, requestedOutput.Name, requestedOutput.Value, expr))
								substitution = "(unknown)"
							}
						}
						if parameters.RequireExpansion(substitution) {
							errs = append(errs, fmt.Errorf("Component `%s` output `%s = %s` refer to substitution `%s` that expands to `%s`. This is surely a bug.",
								componentName, requestedOutput.Name, requestedOutput.Value, expr, substitution))
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

func parseTextOutput(textOutput []byte) parameters.RawOutputs {
	outputs := make(map[string][]string)
	outputsMarker := []byte("Outputs:\n")
	chunk := 1
	for {
		i := bytes.Index(textOutput, outputsMarker)
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
		i = bytes.Index(textFragment, []byte("\n\n"))
		if i > 0 {
			textFragment = textFragment[:i]
		}
		if config.Trace {
			log.Printf("Parsing output chunk #%d:\n%s", chunk, textFragment)
			chunk++
		}
		lines := strings.Split(string(textFragment), "\n")
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
