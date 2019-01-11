package state

import (
	"time"

	"hub/parameters"
)

type Metadata struct {
	Kind string `yaml:",omitempty" json:"kind,omitempty"`
	Name string `yaml:",omitempty" json:"name,omitempty"`
}

type Lifecycle struct {
	Order []string `yaml:",omitempty"`
}

type StateStep struct {
	Timestamp       time.Time
	Parameters      []parameters.LockedParameter `yaml:",omitempty"`
	RawOutputs      []parameters.RawOutput       `yaml:"rawOutputs,omitempty"`
	CapturedOutputs []parameters.CapturedOutput  `yaml:"capturedOutputs,omitempty"`
}

type StateManifest struct {
	Version         int
	Kind            string
	Timestamp       time.Time
	Meta            Metadata                     `yaml:",omitempty"`
	Lifecycle       Lifecycle                    `yaml:",omitempty"`
	StackParameters []parameters.LockedParameter `yaml:"stackParameters,omitempty"`
	CapturedOutputs []parameters.CapturedOutput  `yaml:"capturedOutputs,omitempty"`
	StackOutputs    []parameters.ExpandedOutput  `yaml:"stackOutputs,omitempty"`
	Provides        map[string][]string          `yaml:",omitempty"`
	Components      map[string]StateStep         `yaml:",omitempty"`
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
