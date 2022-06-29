// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

type Request struct {
	Verb                       string
	DryRun                     bool
	ManifestFilenames          []string
	StateFilenames             []string
	LoadFinalState             bool
	EnabledClouds              []string
	Component                  string   // invoke
	Components                 []string // deploy & undeploy, backup
	OffsetComponent            string   // deploy & undeploy
	LimitComponent             string   // deploy & undeploy
	GuessComponent             bool     // undeploy
	OsEnvironmentMode          string
	EnvironmentOverrides       string
	ComponentsBaseDir          string
	GitOutputs                 bool
	GitOutputsStatus           bool
	Environment                string
	StackInstance              string
	Application                string
	SyncStackInstance          bool
	SyncSkipParametersAndOplog bool
	WriteOplogToStateOnError   bool
}
