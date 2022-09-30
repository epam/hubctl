// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package state

import (
	"log"
	"sort"

	"github.com/epam/hubctl/cmd/hub/parameters"
	"github.com/epam/hubctl/cmd/hub/util"
)

func printStateComponents(m map[string]*StateStep) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, name := range keys {
		log.Printf("\t%s", name)
	}
}

func printState(state *StateManifest) {
	if len(state.Components) > 0 {
		log.Print("State components:")
		printStateComponents(state.Components)
	}
	if len(state.StackParameters) > 0 {
		log.Print("State stack parameters:")
		parameters.PrintLockedParametersList(state.StackParameters)
	}
	if len(state.CapturedOutputs) > 0 {
		log.Print("State captured outputs:")
		parameters.PrintCapturedOutputsList(state.CapturedOutputs)
	}
	if len(state.StackOutputs) > 0 {
		log.Print("State stack outputs:")
		for _, stackOutput := range state.StackOutputs {
			log.Printf("\t%s = %s", stackOutput.Name, stackOutput.Value)
		}
	}
	if len(state.Provides) > 0 {
		log.Print("State provides:")
		util.PrintDeps(state.Provides)
	}
}
