// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/util"
)

type ExplainedComponent struct {
	Timestamp  time.Time         `yaml:",omitempty" json:"timestamp,omitempty"`
	Timestamps Timestamps        `yaml:",omitempty" json:"timestamps,omitempty"`
	Status     string            `yaml:",omitempty" json:"status,omitempty"`
	Message    string            `yaml:",omitempty" json:"message,omitempty"`
	Parameters map[string]string `yaml:",omitempty" json:"parameters,omitempty"`
	Outputs    map[string]string `yaml:",omitempty" json:"outputs,omitempty"`
	RawOutputs map[string]string `yaml:"rawOutputs,omitempty" json:"rawOutputs,omitempty"`
}

type ExplainedState struct {
	Meta            Metadata                      `yaml:",omitempty" json:"meta,omitempty"`
	Timestamp       time.Time                     `yaml:",omitempty" json:"timestamp,omitempty"`
	Status          string                        `yaml:",omitempty" json:"status,omitempty"`
	Message         string                        `yaml:",omitempty" json:"message,omitempty"`
	StackParameters map[string]string             `yaml:"stackParameters,omitempty" json:"stackParameters,omitempty"`
	StackOutputs    map[string]string             `yaml:"stackOutputs,omitempty" json:"stackOutputs,omitempty"`
	Provides        map[string][]string           `yaml:",omitempty" json:"provides,omitempty"`
	Components      map[string]ExplainedComponent `yaml:",omitempty" json:"components,omitempty"`
}

func Explain(elaborateManifests, stateFilenames []string, opLog, global bool, componentName string, rawOutputs bool,
	format string /*text, kv, sh, json, yaml*/, color bool) {

	if (color || config.Tty) && format == "text" {
		headColor = func(str string) string {
			return aurora.Green(str).String()
		}
	}

	if format != "text" && config.Verbose && !config.Debug {
		config.Verbose = false
	}

	if opLog && format != "text" {
		log.Fatal("Lifecycle operations log can only be explained in text format")
	}

	state := MustParseStateFiles(stateFilenames)
	components := state.Lifecycle.Order

	if opLog {
		printOpLog(state)
		return
	}

	var stackManifest *manifest.Manifest
	if len(elaborateManifests) > 0 {
		var err error
		stackManifest, _, _, err = manifest.ParseManifest(elaborateManifests)
		if err != nil {
			util.Warn("Unable to parse: %v", err)
		} else if stackManifest != nil {
			components = stackManifest.Lifecycle.Order
		}
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
			fmt.Printf("Timestamp: %v\n", state.Timestamp.Truncate(time.Second))
			fmt.Printf("Status: %s\n", state.Status)
			if state.Message != "" {
				fmt.Printf("Message: %s\n", state.Message)
			}
			fmt.Print(headColor("Stack parameters:\n"))
			printLockedParameters(state.StackParameters)
			printStackOutputs(state.StackOutputs)
			printProvides(state.Provides)
		}

		if !global || componentName != "" {
			for _, component := range components {
				if step, exist := state.Components[component]; exist {
					fmt.Printf("Component: %s\n", headColor(component))
					printComponenentState(component, step, prevOutputs, rawOutputs)
					prevOutputs = step.CapturedOutputs
				}
			}
		}
	} else {
		explained := ExplainedState{
			Meta:            state.Meta,
			Timestamp:       state.Timestamp,
			Status:          state.Status,
			Message:         state.Message,
			StackParameters: make(map[string]string),
			StackOutputs:    make(map[string]string),
			Components:      make(map[string]ExplainedComponent),
		}

		if global || componentName == "" {
			for _, parameter := range state.StackParameters {
				explained.StackParameters[parameter.QName()] = util.String(parameter.Value)
			}
			for _, output := range state.StackOutputs {
				explained.StackOutputs[output.Name] = util.String(output.Value)
			}
			explained.Provides = state.Provides
		}

		if !global || componentName != "" {
			for _, component := range components {
				if step, exist := state.Components[component]; exist {
					comp := ExplainedComponent{
						Timestamp:  step.Timestamp,
						Timestamps: step.Timestamps,
						Status:     step.Status,
						Message:    step.Message,
						Parameters: make(map[string]string),
						Outputs:    make(map[string]string),
						RawOutputs: make(map[string]string),
					}
					for _, parameter := range step.Parameters {
						comp.Parameters[parameter.Name] = util.String(parameter.Value)
					}
					for _, output := range DiffOutputs(step.CapturedOutputs, prevOutputs) {
						comp.Outputs[output.Name] = util.String(output.Value)
					}
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

func printComponenentState(componentName string, step *StateStep, prevOutputs []parameters.CapturedOutput, rawOutputs bool) {
	fmt.Printf("-- Timestamp: %v\n", step.Timestamp.Truncate(time.Second))
	if t := step.Timestamps; !t.End.IsZero() && !t.Start.IsZero() {
		fmt.Printf("-- Duration: %v\n", t.End.Sub(t.Start).Round(time.Second).String())
	}
	fmt.Printf("-- Status: %s\n", step.Status)
	if step.Meta.Origin != "" && step.Meta.Origin != componentName {
		fmt.Printf("-- Origin: %s\n", step.Meta.Origin)
	}
	if step.Meta.Kind != "" && step.Meta.Kind != step.Meta.Origin {
		fmt.Printf("-- Kind: %s\n", step.Meta.Kind)
	}
	if step.Meta.Title != "" {
		fmt.Printf("-- Title: %s\n", step.Meta.Title)
	}
	version := step.Meta.Version
	if version == "" && step.Version != "" {
		version = step.Version
	}
	if version != "" {
		fmt.Printf("-- Version: %s\n", version)
	}
	if step.Message != "" {
		fmt.Printf("-- Message: %s\n", step.Message)
	}
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
		fmt.Printf("\t%s => `%s`%s\n", qName, util.Wrap(util.String(parameter.Value)), env)
	}
}

func printDiffOutputs(curr, prev []parameters.CapturedOutput) {
	keys := make(map[string]string)
	for _, p := range prev {
		str := util.String(p.Value)
		keys[p.QName()] = str
		keys[p.Name] = str
	}
	for _, c := range curr {
		if strings.HasPrefix(c.Name, "hub.components.") {
			continue
		}
		qName := c.QName()
		_, exist := keys[qName]
		if !exist {
			over, overExist := keys[c.Name]
			kind := ""
			if c.Kind != "" {
				kind = fmt.Sprintf("[%s] ", c.Kind)
			}
			brief := ""
			if c.Brief != "" {
				brief = fmt.Sprintf(" [%s]", c.Brief)
			}
			value := util.Wrap(util.String(c.Value))
			if !overExist {
				fmt.Printf("\t%s%s%s => `%s`\n", kind, c.Name, brief, value)
			} else if util.String(c.Value) != over {
				fmt.Printf("\t%s%s%s => `%s` (was: `%s`)\n", kind, c.Name, brief, value, util.Wrap(over))
			} else {
				fmt.Printf("\t%s%s%s => `%s`\n", kind, qName, brief, value)
			}
		}
	}
}

func DiffOutputs(curr, prev []parameters.CapturedOutput) []parameters.CapturedOutput {
	keys := make(map[string]struct{})
	for _, p := range prev {
		keys[p.QName()] = struct{}{}
	}
	sz := len(curr) - len(prev)
	if sz < 0 {
		sz = 0
	}
	diff := make([]parameters.CapturedOutput, 0, sz)
	for _, c := range curr {
		if strings.HasPrefix(c.Name, "hub.components.") {
			continue
		}
		if _, exist := keys[c.QName()]; !exist {
			diff = append(diff, c)
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
			kind := ""
			if expandedOutput.Kind != "" {
				kind = fmt.Sprintf("[%s] ", expandedOutput.Kind)
			}
			fmt.Printf("\t%s%s%s = %s\n", brief, kind, expandedOutput.Name, expandedOutput.Value)
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

func printOpLog(st *StateManifest) {
	ops := st.Operations
	if len(ops) == 0 {
		fmt.Print("No operations log")
	}
	fmt.Print("Operations:\n")
	for _, op := range ops {
		fmt.Print(formatOperation(op, true))
	}
}

func formatOperation(op LifecycleOperation, showLogs bool) string {
	ident := "\t"
	logs := ""
	if showLogs && op.Logs != "" {
		logs = fmt.Sprintf("%sLogs:\n%s\t%s\n",
			ident, ident, strings.Join(strings.Split(op.Logs, "\n"), "\n"+ident+"\t"))
	}
	initiator := ""
	if op.Initiator != "" {
		initiator = fmt.Sprintf(" by %s", op.Initiator)
	}
	options := ""
	if len(op.Options) > 0 {
		options = fmt.Sprintf("%sOptions: %v\n", ident, op.Options)
	}
	description := ""
	if op.Description != "" {
		description = fmt.Sprintf(" (%s)", op.Description)
	}
	phases := ""
	if len(op.Phases) > 0 {
		phases = fmt.Sprintf("%sPhases:\n%s\t%s\n", ident, ident, formatLifecyclePhases(op.Phases, ident))
	}
	return fmt.Sprintf("%s%s %s - %s %v%s%s %s\n%s%s%s",
		ident, headColor("Operation:"), op.Operation, op.Status, op.Timestamp.Truncate(time.Second), initiator, description, op.Id,
		options, phases, logs)
}

func formatLifecyclePhases(phases []LifecyclePhase, ident string) string {
	str := make([]string, 0, len(phases))
	for _, phase := range phases {
		str = append(str, fmt.Sprintf("%s - %s", phase.Phase, phase.Status))
	}
	return strings.Join(str, "\n"+ident+"\t")
}
