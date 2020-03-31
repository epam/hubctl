package state

import (
	"fmt"
	"log"
	"strings"
	"time"

	"hub/config"
	"hub/parameters"
	"hub/storage"
	"hub/util"
)

func MergeState(stateFiles *storage.Files,
	componentName string, depends []string, order []string, isDeploy bool,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs, provides map[string][]string) (*StateManifest, error) {

	state, err := ParseState(stateFiles)
	if err != nil {
		return nil, err
	}
	MergeParsedState(state, componentName, depends, order, isDeploy, parameters, outputs, provides)
	return state, nil
}

func MergeParsedState(state *StateManifest,
	componentName string, depends []string, order []string, isDeploy bool,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs, provides map[string][]string) {

	MergeParsedStateParametersAndProvides(state, parameters, provides)
	MergeParsedStateOutputs(state, componentName, depends, order, isDeploy, outputs)
}

func MergeParsedStateParametersAndProvides(state *StateManifest,
	parameters parameters.LockedParameters, provides map[string][]string) {

	if parameters != nil {
		mergeStateParameters(parameters, state.StackParameters)
	}
	if provides != nil {
		mergeStateProvides(provides, state.Provides)
	}
}

func MergeParsedStateOutputs(state *StateManifest,
	componentName string, depends []string, order []string, isDeploy bool,
	outputs parameters.CapturedOutputs) {

	if outputs != nil {
		outputsToMerge, mergedTimestamp := chooseStateOutputsToMerge(state, componentName, order, isDeploy)
		mergeStateOutputs(outputs, outputsToMerge)
		mergeStateOutputsFromDependencies(outputs, depends, mergedTimestamp, state.Components)
	}
}

func chooseStateOutputsToMerge(state *StateManifest, componentName string, order []string,
	isDeploy bool) ([]parameters.CapturedOutput, time.Time) {

	outputsToMerge := state.CapturedOutputs
	mergedTimestamp := state.Timestamp

	componentStateName := ""
	componentStateExist := false
	var componentState *StateStep
	if state.Components != nil {
		if componentName == "" && len(outputsToMerge) == 0 {
			componentName = order[len(order)-1] // if global state is empty then start search with the last component
		}

		if componentName != "" {
			found := -1
			for i, name := range order {
				if componentName == name {
					found = i
					break
				}
			}
			if isDeploy {
				if found == 0 {
					// first component is a special case
					outputsToMerge = nil
				}
				found--
			}
			for found >= 0 {
				componentStateName = order[found]
				st, exist := state.Components[componentStateName]
				if exist && len(st.CapturedOutputs) > 0 {
					break
				}
				found--
			}
			if found < 0 {
				componentStateName = ""
			}
		}

		if componentStateName != "" {
			componentState, componentStateExist = state.Components[componentStateName]
		}
	}

	if componentStateExist {
		outputsToMerge = componentState.CapturedOutputs
		mergedTimestamp = componentState.Timestamp
		if config.Verbose {
			log.Printf("Loading state after component `%s` deployment", componentStateName)
		}
	} else {
		if config.Verbose && outputsToMerge != nil {
			log.Print("Loading global state")
		}
	}

	if len(outputsToMerge) == 0 && config.Verbose && outputsToMerge != nil {
		label := "Global"
		hint := ""
		if componentStateExist {
			label = fmt.Sprintf("Component `%s`", componentStateName)
			hint = " (try --load-global-state / -g)"
		}
		log.Printf("%s outputs state is empty%s ", label, hint)
	}

	return outputsToMerge, mergedTimestamp
}

func mergeStateParameters(parameters parameters.LockedParameters, add []parameters.LockedParameter) {
	for _, p := range add {
		mergeStateParameter(parameters, p)
	}
}

func mergeStateParameter(parameters parameters.LockedParameters, add parameters.LockedParameter) {
	qName := add.QName()
	current, exists := parameters[qName]
	if exists {
		curValue := util.String(current.Value)
		addValue := util.String(add.Value)
		if curValue != addValue {
			if util.Empty(current.Value) {
				util.Warn("Parameter `%s` empty value is replaced by value `%s` from state",
					qName, util.Trim(addValue))
				current.Value = add.Value
			} else {
				util.Warn("Parameter `%s` current value `%s` does not match value `%s` from state - keeping current value",
					qName, util.Trim(curValue), util.Trim(addValue))
			}
		}
		if current.Env != add.Env {
			if current.Env == "" {
				if config.Debug {
					log.Printf("Parameter `%s` environment variable setup is updated to `%s` from state",
						qName, add.Env)
				}
				current.Env = add.Env
			} else if add.Env != "" {
				util.Warn("Parameter `%s` current environment variable setup `%s` does not match setup `%s` from state - keeping current setup",
					qName, current.Env, add.Env)
			}
		}
		parameters[qName] = current
	} else {
		// TODO review: should we really merge stack-level parameters from state?
		parameters[qName] = add
	}
}

func mergeStateOutputs(outputs parameters.CapturedOutputs, state []parameters.CapturedOutput) {
	for _, o := range state {
		parameters.MergeOutput(outputs, o)
	}
}

func mergeStateOutputsFromDependencies(outputs parameters.CapturedOutputs, depends []string, mergedTimestamp time.Time,
	components map[string]*StateStep) {

	loading := make([]string, 0, len(depends))
	for _, dependencyName := range depends {
		// TODO review: always load outputs from dependencies?
		if _, exist := components[dependencyName]; exist /* && dependency.Timestamp.After(mergedTimestamp) */ {
			loading = append(loading, dependencyName)
		}
	}
	if config.Verbose && len(loading) > 0 {
		log.Printf("Additionally, loading state after component(s) %v due to `depends` declaration", loading)
	}
	for _, dependencyName := range util.Reverse(loading) {
		if dependency, exist := components[dependencyName]; exist {
			for _, output := range dependency.CapturedOutputs {
				qName := output.QName()
				current, exists := outputs[qName]
				overwrite := false
				if exists {
					warn := !strings.HasPrefix(output.Name, "hub.")
					if util.String(current.Value) != util.String(output.Value) {
						if !util.Empty(current.Value) {
							overwrite = true // TODO review overwrite logic
							if warn {
								util.Warn("Loaded output `%s` current value `%v` does not match new value `%v` loaded from dependency `%s` - using new value",
									qName, current.Value, output.Value, dependencyName)
							}
						} else if !util.Empty(output.Value) {
							overwrite = true
							if warn {
								util.Warn("Loaded output `%s` empty value replaced by new value `%v` loaded from dependency `%s`",
									qName, output.Value, dependencyName)
							}
						}
					}
				} else {
					overwrite = true
				}
				if overwrite {
					if config.Debug {
						log.Printf("\t%s => %s", qName, output.Value)
					}
					outputs[qName] = output
				}
			}
		}
	}
}

func mergeStateProvides(provides map[string][]string, state map[string][]string) {
	for prov, by := range state {
		who, exist := provides[prov]
		if !exist {
			provides[prov] = by
		} else {
			merged := make([]string, len(who), len(who)+len(by))
			copy(merged, who)
			for _, comp := range by {
				found := false
				for _, comp2 := range who {
					if comp == comp2 {
						found = true
						break
					}
				}
				if !found {
					merged = append(merged, comp)
				}
			}
			provides[prov] = merged
		}
	}
}
