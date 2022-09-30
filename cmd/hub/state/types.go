// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package state

import (
	"time"

	"github.com/epam/hubctl/cmd/hub/parameters"
)

type Metadata struct {
	Kind string `yaml:",omitempty" json:"kind,omitempty"`
	Name string `yaml:",omitempty" json:"name,omitempty"`
}

type Lifecycle struct {
	Order []string `yaml:",omitempty"`
}

type Timestamps struct {
	Start time.Time `yaml:",omitempty" json:"start,omitempty"`
	End   time.Time `yaml:",omitempty" json:"end,omitempty"`
}

type ComponentMetadata struct {
	Origin      string `yaml:",omitempty"`
	Kind        string `yaml:",omitempty"`
	Title       string `yaml:",omitempty"`
	Brief       string `yaml:",omitempty"`
	Description string `yaml:",omitempty"`
	Version     string `yaml:",omitempty"`
	Maturity    string `yaml:",omitempty"`
	Icon        string `yaml:",omitempty"`
}

type StateStep struct {
	Timestamp       time.Time                    `yaml:",omitempty"`
	Timestamps      Timestamps                   `yaml:",omitempty"`
	Status          string                       `yaml:",omitempty"`
	Version         string                       `yaml:",omitempty"` // TODO deprecate in favor of Meta
	Meta            ComponentMetadata            `yaml:",omitempty"`
	Message         string                       `yaml:",omitempty"`
	Parameters      []parameters.LockedParameter `yaml:",omitempty"`
	RawOutputs      []parameters.RawOutput       `yaml:"rawOutputs,omitempty"`
	CapturedOutputs []parameters.CapturedOutput  `yaml:"capturedOutputs,omitempty"`
}

type LifecyclePhase struct {
	Phase  string `yaml:",omitempty"`
	Status string `yaml:",omitempty"`
}

type LifecycleOperation struct {
	Id          string
	Operation   string
	Timestamp   time.Time
	Status      string                 `yaml:",omitempty"`
	Options     map[string]interface{} `yaml:",omitempty"`
	Description string                 `yaml:",omitempty"`
	Initiator   string                 `yaml:",omitempty"`
	Logs        string                 `yaml:",omitempty"`
	Phases      []LifecyclePhase       `yaml:",omitempty"`
}

type StateManifest struct {
	Version         int
	Kind            string
	Timestamp       time.Time
	Status          string                       `yaml:",omitempty"`
	Message         string                       `yaml:",omitempty"`
	Meta            Metadata                     `yaml:",omitempty"`
	Lifecycle       Lifecycle                    `yaml:",omitempty"`
	StackParameters []parameters.LockedParameter `yaml:"stackParameters,omitempty"`
	CapturedOutputs []parameters.CapturedOutput  `yaml:"capturedOutputs,omitempty"`
	StackOutputs    []parameters.ExpandedOutput  `yaml:"stackOutputs,omitempty"`
	Provides        map[string][]string          `yaml:",omitempty"`
	Components      map[string]*StateStep        `yaml:",omitempty"`
	Operations      []LifecycleOperation         `yaml:",omitempty"`
}

type ComponentBackup struct {
	Timestamp time.Time                   `json:"timestamp"`
	Status    string                      `json:"status"`
	Kind      string                      `json:"kind"`
	Outputs   []parameters.CapturedOutput `yaml:",omitempty" json:"outputs,omitempty"`

	// not present in backup bundle, used for diagnostic in `backup unbundle`
	Source    string `yaml:",omitempty" json:"source,omitempty"`
	FileIndex int    `yaml:",omitempty" json:"fileIndex,omitempty"`
}

type BackupManifest struct {
	Version    int                        `json:"version"`
	Kind       string                     `json:"kind"`
	Timestamp  time.Time                  `json:"timestamp"`
	Status     string                     `json:"status"`
	Components map[string]ComponentBackup `json:"components"`

	// not present in backup bundle, used for diagnostic in `backup unbundle`
	Source string `yaml:",omitempty" json:"source,omitempty"`
}
