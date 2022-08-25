// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package cmd

const (
	SuperHubIo = ".superhub.io"

	envVarNameHubApi       = "HUB_API"
	envVarNameDerefSecrets = "HUB_API_DEREF_SECRETS"

	mdpre = "```"
)

var (
	supportedCloudAccountKinds = []string{"aws", "azure", "gcp"}
)

var (
	environmentSelector   string
	templateSelector      string
	waitAndTailDeployLogs bool
	showSecrets           bool
	showLogs              bool
	jsonFormat            bool
	patchReplace          bool
	patchRaw              bool
	createRaw             bool
)
