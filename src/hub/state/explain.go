package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

type ExplainedComponent struct {
	Timestamp  time.Time         `yaml:",omitempty" json:"timestamp,omitempty"`
	Parameters map[string]string `yaml:",omitempty" json:"parameters,omitempty"`
	Outputs    map[string]string `yaml:",omitempty" json:"outputs,omitempty"`
	RawOutputs map[string]string `yaml:"rawOutputs,omitempty" json:"rawOutputs,omitempty"`
}

type ExplainedState struct {
	Meta            Metadata                      `yaml:",omitempty" json:"meta,omitempty"`
	Timestamp       time.Time                     `yaml:",omitempty" json:"timestamp,omitempty"`
	StackParameters map[string]string             `yaml:"stackParameters,omitempty" json:"stackParameters,omitempty"`
	StackOutputs    map[string]string             `yaml:"stackOutputs,omitempty" json:"stackOutputs,omitempty"`
	Provides        map[string][]string           `yaml:",omitempty" json:"provides,omitempty"`
	Components      map[string]ExplainedComponent `yaml:",omitempty" json:"components,omitempty"`
}

func Explain(elaborateManifests, stateFilenames []string, global bool, componentName string, rawOutputs bool,
	format string /*text, kv, sh, json, yaml*/, color bool) {

	if color && format == "text" {
		headColor = func(str string) string {
			return aurora.Green(str).String()
		}
	}

	if format != "text" && config.Verbose && !config.Debug {
		config.Verbose = false
	}

	state := MustParseStateFiles(stateFilenames)
	components := state.Lifecycle.Order

	var stackManifest *manifest.Manifest
	if len(elaborateManifests) > 0 {
		var err error
		stackManifest, _, _, err = manifest.ParseManifest(elaborateManifests)
		if err != nil {
			log.Fatalf("Unable to parse: %v", err)
		}
		components = stackManifest.Lifecycle.Order
	}

	var prevOutputs []parameters.CapturedOutput

	if componentName != "" {
		if stackManifest != nil {
			manifest.CheckComponentsExist(stackManifest.Components, componentName)
		}

		for i, c := range components {
			if c == componentName {
				if i > 0 {
					prevComponentState, exist := state.Components[components[i-1]]
					if exist {
						prevOutputs = prevComponentState.CapturedOutputs
					}
				}
			}
		}

		components = []string{componentName}
	}

	if format == "text" {
		if global || componentName == "" {
			fmt.Printf("Kind: %s\n", state.Meta.Kind)
			fmt.Printf("Name: %s\n", state.Meta.Name)
			fmt.Printf("Timestamp: %v\n", state.Timestamp)
			fmt.Print(headColor("Stack parameters:\n"))
			printLockedParameters(state.StackParameters)
			printStackOutputs(state.StackOutputs)
			printProvides(state.Provides)
		}

		if !global || componentName != "" {
			for _, component := range components {
				if step, exist := state.Components[component]; exist {
					fmt.Printf("Component: %s\n", headColor(component))
					printComponenentState(step, prevOutputs, rawOutputs)
					prevOutputs = step.CapturedOutputs
				}
			}
		}
	} else {
		explained := ExplainedState{
			Meta:            state.Meta,
			Timestamp:       state.Timestamp,
			StackParameters: make(map[string]string),
			StackOutputs:    make(map[string]string),
			Components:      make(map[string]ExplainedComponent),
		}

		if global || componentName == "" {
			for _, parameter := range state.StackParameters {
				explained.StackParameters[parameter.QName()] = parameter.Value
			}
			for _, output := range state.StackOutputs {
				explained.StackOutputs[output.Name] = output.Value
			}
			explained.Provides = state.Provides
		}

		if !global || componentName != "" {
			for _, component := range components {
				if step, exist := state.Components[component]; exist {
					comp := ExplainedComponent{
						Timestamp:  step.Timestamp,
						Parameters: make(map[string]string),
						RawOutputs: make(map[string]string),
					}
					for _, parameter := range step.Parameters {
						comp.Parameters[parameter.Name] = parameter.Value
					}
					comp.Outputs = diffOutputs(step.CapturedOutputs, prevOutputs)
					prevOutputs = step.CapturedOutputs
					if rawOutputs {
						for _, output := range step.RawOutputs {
							comp.RawOutputs[output.Name] = output.Value
						}
					}
					explained.Components[component] = comp
				}
			}
		}

		var bytes []byte
		var err error

		switch format {
		case "json":
			bytes, err = json.MarshalIndent(&explained, "", "  ")
		case "yaml":
			bytes, err = yaml.Marshal(&explained)
		// case "sh":
		default:
			log.Fatalf("`%s` output format is not implemented", format)
		}

		if err != nil {
			log.Fatalf("Unable to explain in `%s` format: %v", format, err)
		}

		written, err := os.Stdout.Write(bytes)
		if err != nil || written != len(bytes) {
			log.Fatalf("Error writting output (wrote %d of ouf %d bytes): %v", written, len(bytes), err)
		}
	}
}

var headColor = func(str string) string {
	return str
}

func printComponenentState(step StateStep, prevOutputs []parameters.CapturedOutput, rawOutputs bool) {
	fmt.Printf("-- Timestamp: %v\n", step.Timestamp)
	fmt.Print("-- Parameters:\n")
	printLockedParameters(step.Parameters)
	if rawOutputs && len(step.RawOutputs) > 0 {
		fmt.Print("-- Raw outputs:\n")
		printRawOutputs(step.RawOutputs)
	}
	fmt.Print("-- Outputs:\n")
	printDiffOutputs(step.CapturedOutputs, prevOutputs)
}

func printLockedParameters(parameters []parameters.LockedParameter) {
	for _, parameter := range parameters {
		qName := parameter.QName()
		env := ""
		if parameter.Env != "" {
			env = fmt.Sprintf(" (env:%s)", parameter.Env)
		}
		fmt.Printf("\t%s => `%s`%s\n", qName, util.Wrap(parameter.Value), env)
	}
}

func printDiffOutputs(curr, prev []parameters.CapturedOutput) {
	keys := make(map[string]string)
	for _, p := range prev {
		qName := p.QName()
		keys[qName] = p.Value
		keys[p.Name] = p.Value
	}
	for _, c := range curr {
		if strings.HasPrefix(c.Name, "hub.components.") {
			continue
		}
		qName := c.QName()
		_, exist := keys[qName]
		if !exist {
			over, overExist := keys[c.Name]
			if !overExist {
				fmt.Printf("\t%s => %s\n", c.Name, util.Wrap(c.Value))
			} else if c.Value != over {
				fmt.Printf("\t%s => %s (was: %s)\n", c.Name, util.Wrap(c.Value), util.Wrap(over))
			}
		}
	}
}

func diffOutputs(curr, prev []parameters.CapturedOutput) map[string]string {
	keys := make(map[string]string)
	for _, p := range prev {
		keys[p.QName()] = p.Value
	}
	diff := make(map[string]string)
	for _, c := range curr {
		if strings.HasPrefix(c.Name, "hub.components.") {
			continue
		}
		if _, exist := keys[c.QName()]; !exist {
			diff[c.Name] = c.Value
		}
	}
	return diff
}

func printRawOutputs(rawOutputs []parameters.RawOutput) {
	for _, o := range rawOutputs {
		fmt.Printf("\t%s = %s\n", o.Name, o.Value)
	}
}

func printStackOutputs(expanded []parameters.ExpandedOutput) {
	if len(expanded) > 0 {
		fmt.Print(headColor("Stack outputs:\n"))
		for _, expandedOutput := range expanded {
			brief := ""
			if expandedOutput.Brief != "" {
				brief = fmt.Sprintf("[%s] ", expandedOutput.Brief)
			}
			fmt.Printf("\t%s%s = %s\n", brief, expandedOutput.Name, expandedOutput.Value)
		}
	}
}

func printProvides(deps map[string][]string) {
	if len(deps) > 0 {
		fmt.Print(headColor("Provides:\n"))
		keys := make([]string, 0, len(deps))
		for name := range deps {
			keys = append(keys, name)
		}
		sort.Strings(keys)

		for _, name := range keys {
			fmt.Printf("\t%s => %s\n", name, strings.Join(deps[name], ", "))
		}
	}
}
