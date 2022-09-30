// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package lifecycle

import (
	"fmt"
	"log"
	"strings"

	"github.com/epam/hubctl/cmd/hub/api"
	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/state"
	"github.com/epam/hubctl/cmd/hub/storage"
	"github.com/epam/hubctl/cmd/hub/util"
)

func hubSyncer(request *Request) func(*state.StateManifest) {
	return func(stateManifest *state.StateManifest) {
		patch := api.TransformStateToApi(stateManifest)
		remoteStatePaths := storage.RemoteStoragePaths(request.StateFilenames)
		if len(remoteStatePaths) > 0 {
			patch.StateFiles = remoteStatePaths
		}
		if request.SyncSkipParametersAndOplog {
			patch.ComponentsEnabled = nil
			patch.Parameters = nil
			patch.InflightOperations = nil
		}
		if config.Verbose {
			log.Print("Syncing Stack Instance state to HubCTL")
			if config.Trace {
				printStackInstancePatch(patch)
			}
		}
		_, err := api.PatchStackInstance(request.StackInstance, patch, true)
		if err != nil {
			util.Warn("Unable to sync stack instance state to HubCTL: %v\n\ttry running sync manually: hub api instance sync %s -s %s ",
				err, request.StackInstance, strings.Join(request.StateFilenames, ","))
		}
	}
}

func printStackInstancePatch(patch api.StackInstancePatch) {
	if len(patch.Outputs) > 0 {
		log.Print("Outputs to API:")
		for _, output := range patch.Outputs {
			brief := ""
			if output.Brief != "" {
				brief = fmt.Sprintf("[%s] ", output.Brief)
			}
			component := ""
			if output.Component != "" {
				component = fmt.Sprintf("%s:", output.Component)
			}
			// this is under Trace, no secret value masking required
			log.Printf("\t%s%s%s => `%v`", brief, component, output.Name, output.Value)
		}
	}
	if len(patch.Provides) > 0 {
		log.Print("Provides to API:")
		util.PrintMap2(patch.Provides)
	}
}
