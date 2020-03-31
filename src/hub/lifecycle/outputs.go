package lifecycle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

const fileRefPrefix = "file://"

var (
	outputsMarker            = []byte("Outputs:\n")
	outputSupportedEncodings = []string{"base64", "json"}
)

func captureOutputs(componentName, componentDir string, componentManifest *manifest.Manifest,
	componentParameters parameters.LockedParameters,
	textOutput []byte, random []byte) (parameters.RawOutputs, parameters.CapturedOutputs, []string, []error) {

	tfOutputs := parseTextOutput(textOutput)
	secrets := extractSecrets(tfOutputs, random)
	if len(secrets) > 0 {
		if config.Trace {
			log.Printf("Parsing secrets chunk:\n%s", secrets)
		}
		outputs := make(map[string][]string)
		parseTextKV(secrets, outputs)
		if config.Trace {
			log.Print("Parsed secret outputs:")
			util.PrintMap2(outputs)
		}
		for k, v := range toRawOutputs(outputs) {
			tfOutputs[k] = v
		}
	}
	dynamicProvides := extractDynamicProvides(tfOutputs)
	outputs, errs := expandRequestedOutputs(componentName, componentDir, componentParameters, componentManifest.Outputs, tfOutputs)
	for k, o := range outputs {
		o.Component = componentName
		o.ComponentOrigin = componentManifest.Meta.Origin
		o.ComponentKind = componentManifest.Meta.Kind
		outputs[k] = o
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

func expandRequestedOutputs(componentName, componentDir string,
	componentParameters parameters.LockedParameters, requestedOutputs []manifest.Output,
	tfOutputs parameters.RawOutputs) (parameters.CapturedOutputs, []error) {

	kv := parameters.ParametersKV(componentParameters)
	outputs := make(parameters.CapturedOutputs)
	errs := make([]error, 0)
	for _, requestedOutput := range requestedOutputs {
		output := parameters.CapturedOutput{
			Name:  requestedOutput.Name,
			Brief: requestedOutput.Brief,
			Kind:  requestedOutput.Kind,
		}
		if requestedOutput.FromTfVar != "" {
			variable, encodings := valueEncodings(requestedOutput.FromTfVar)
			value, exist := tfOutputs[variable]
			if !exist {
				errs = append(errs, fmt.Errorf("Unable to capture raw output `%s` for component `%s` output `%s`",
					variable, componentName, requestedOutput.Name))
				value = "(unknown)"
			} else {
				if strings.HasPrefix(value, fileRefPrefix) && len(value) > len(fileRefPrefix) {
					filename := value[len(fileRefPrefix):]
					if !filepath.IsAbs(filename) {
						filename = filepath.Join(componentDir, filename)
					}
					bytes, err := ioutil.ReadFile(filename)
					if err != nil {
						util.Warn("Unable to read raw output `%s` from `%s` for component `%s` output `%s`: %v",
							variable, filename, componentName, requestedOutput.Name, err)
						// pass value as is
					} else {
						value = string(bytes)
						if strings.Count(value, "\n") <= 1 {
							value = strings.Trim(value, " \r\n")
						}
					}
				}
				if len(encodings) > 0 {
					if unknown := util.OmitAll(encodings, outputSupportedEncodings); len(unknown) > 0 {
						errs = append(errs, fmt.Errorf("Unknown encoding(s) %v capturing output `%s` from raw output `%s`",
							unknown, requestedOutput.FromTfVar, variable))
					}
					if util.Contains(encodings, "base64") {
						bValue, err := base64.StdEncoding.DecodeString(value)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to decode base64 `%s` while capturing output `%s` from raw output `%s`: %v",
								util.Trim(value), requestedOutput.FromTfVar, variable, err))
						} else {
							value = string(bValue)
						}
					}
					if util.Contains(encodings, "json") {
						var iValue interface{}
						err := json.Unmarshal([]byte(value), &iValue)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to unmarshal JSON `%s` while capturing output `%s` from raw output `%s`: %v",
								util.Trim(value), requestedOutput.FromTfVar, variable, err))
						} else {
							output.Value = iValue
						}
					}
				}
			}
			if output.Value == nil { // TODO decoded nil from JSON null?
				output.Value = value
			}
		} else {
			if util.Empty(requestedOutput.Value) {
				requestedOutput.Value = fmt.Sprintf("${%s}", requestedOutput.Name)
			}
			if parameters.RequireExpansion(requestedOutput.Value) {
				requestedOutputValue := requestedOutput.Value.(string)
				value := parameters.CurlyReplacement.ReplaceAllStringFunc(requestedOutputValue,
					func(match string) string {
						expr, isCel := parameters.StripCurly(match)
						var substitution interface{}
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
									componentName, requestedOutput.Name, requestedOutputValue, expr))
								substitution = "(unknown)"
							}
						}
						if parameters.RequireExpansion(substitution) {
							errs = append(errs, fmt.Errorf("Component `%s` output `%s = %s` refer to substitution `%s` that expands to `%s` - this is surely a bug",
								componentName, requestedOutput.Name, requestedOutputValue, expr, substitution))
							substitution = "(bug)"
						}
						return util.String(substitution)
					})
				output.Value = value
			} else {
				output.Value = requestedOutput.Value
			}
		}
		outputs[output.QName()] = output
		kv[requestedOutput.Name] = output.Value
	}
	return outputs, errs
}

func parseTextOutput(textOutput []byte) parameters.RawOutputs {
	outputs := make(map[string][]string)
	chunk := 1
	for {
		i := bytes.Index(textOutput, outputsMarker)
		if i == -1 {
			if config.Trace && len(outputs) > 0 {
				log.Print("Parsed raw outputs:")
				util.PrintMap2(outputs)
			}
			return toRawOutputs(outputs)
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
		parseTextKV(textFragment, outputs)
	}
}

func parseTextKV(text []byte, outputs map[string][]string) {
	lines := strings.Split(string(text), "\n")
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

func toRawOutputs(outputs map[string][]string) parameters.RawOutputs {
	rawOutputs := make(parameters.RawOutputs)
	for k, list := range outputs {
		rawOutputs[k] = strings.Join(list, ",")
	}
	return rawOutputs
}

func extractDynamicProvides(rawOutputs parameters.RawOutputs) []string {
	key := "provides"
	if v, exist := rawOutputs[key]; exist && len(v) > 0 {
		return strings.Split(v, ",")
	}
	return nil
}

func extractSecrets(rawOutputs parameters.RawOutputs, random []byte) []byte {
	key := "secrets"
	if v, exist := rawOutputs[key]; exist && len(v) > 0 {
		decoded, err := util.OtpDecode(v, random)
		if err != nil {
			util.Warn("Unable to decode secret outputs: %v", err)
			return nil
		}
		return decoded
	}
	return nil
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
