// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"log"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/storage"
	"github.com/agilestacks/hub/cmd/hub/util"
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
			log.Print("Syncing Stack Instance state to SuperHub")
			if config.Trace {
				printStackInstancePatch(patch)
			}
		}
		_, err := api.PatchStackInstance(request.StackInstance, patch, true)
		if err != nil {
			util.Warn("Unable to sync stack instance state to SuperHub: %v\n\ttry running sync manually: hub api instance sync %s -s %s ",
				err, request.StackInstance, strings.Join(request.StateFilenames, ","))
		}
	}
}
