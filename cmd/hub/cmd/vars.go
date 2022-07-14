// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

const (
	envVarNameHubCli            = "HUB"
	envVarNameElaborate         = "HUB_ELABORATE"
	envVarNameState             = "HUB_STATE"
	envVarNameAwsRegion         = "HUB_AWS_REGION"
	envVarNameComponentsBaseDir = "HUB_COMPONENTS_BASEDIR"
	envVarNameHubApi            = "HUB_API"
	envVarNameDerefSecrets      = "HUB_API_DEREF_SECRETS"
	SuperHubIo                  = ".superhub.io"

	mdpre = "```"
)

var (
	supportedClouds            = []string{"aws", "azure", "gcp"}
	supportedCloudAccountKinds = []string{"aws", "azure", "gcp"}
)

var (
	environmentSelector   string
	templateSelector      string
	componentName         string
	componentsBaseDir     string
	elaborateManifest     string
	stateManifest         string
	stateManifestExplicit string
	environmentOverrides  string
	dryRun                bool
	osEnvironmentMode     string
	outputFiles           string
	waitAndTailDeployLogs bool
	showSecrets           bool
	showLogs              bool
	jsonFormat            bool
	patchReplace          bool
	patchRaw              bool
	createRaw             bool
)
