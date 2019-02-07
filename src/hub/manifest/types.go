package manifest

import "fmt"

type Git struct {
	Remote   string
	Ref      string `yaml:",omitempty"`
	SubDir   string `yaml:"subDir,omitempty"`
	LocalDir string `yaml:"localDir,omitempty"`
}

type SourceLocation struct {
	Dir string `yaml:",omitempty"`
	S3  string `yaml:",omitempty"`
	Git Git    `yaml:",omitempty"`
}

type Metadata struct {
	Name        string
	Brief       string            `yaml:",omitempty"`
	Description string            `yaml:",omitempty"`
	Source      SourceLocation    `yaml:",omitempty"`
	FromStack   string            `yaml:"fromStack,omitempty"`
	Annotations map[string]string `yaml:",omitempty"`
}

type PlatformMetadata struct {
	Provides []string `yaml:",omitempty"`
}

type ComponentRef struct {
	Name        string
	Source      SourceLocation    `yaml:",omitempty"`
	Depends     []string          `yaml:",omitempty"`
	Annotations map[string]string `yaml:",omitempty"`
}

type RequiresTuning struct {
	Optional []string `yaml:",omitempty"`
}

type ReadyCondition struct {
	DNS          string `yaml:"dns,omitempty"`
	URL          string `yaml:"url,omitempty"`
	WaitSeconds  int    `yaml:"waitSeconds,omitempty"`
	PauseSeconds int    `yaml:"pauseSeconds,omitempty"`
}

type Lifecycle struct {
	Bare            string           `yaml:",omitempty"`
	Verbs           []string         `yaml:",omitempty"`
	Order           []string         `yaml:",omitempty"`
	Mandatory       []string         `yaml:",omitempty"`
	Optional        []string         `yaml:",omitempty"`
	Requires        RequiresTuning   `yaml:",omitempty"`
	ReadyConditions []ReadyCondition `yaml:"readyConditions,omitempty"`
}

type Output struct {
	Name        string
	Brief       string `yaml:",omitempty"`
	Description string `yaml:",omitempty"`

	Value     string `yaml:",omitempty"`
	FromTfVar string `yaml:"fromTfVar,omitempty"`
	Kind      string `yaml:",omitempty"`
}

type Parameter struct {
	Name        string
	Component   string `yaml:",omitempty"` // target specific component instance
	Kind        string `yaml:",omitempty"`
	Brief       string `yaml:",omitempty"`
	Description string `yaml:",omitempty"`

	// Default interface{} `yaml:",omitempty"`
	// Value   interface{} `yaml:",omitempty"`
	Default string `yaml:",omitempty"`
	Value   string `yaml:",omitempty"`
	Empty   string `yaml:",omitempty"` // "allow"
	FromEnv string `yaml:"fromEnv,omitempty"`
	Env     string `yaml:",omitempty"`

	Parameters []Parameter `yaml:",omitempty"`
}

type TemplateTarget struct {
	Kind        string   `yaml:",omitempty"`
	Directories []string `yaml:",omitempty"`
	Files       []string `yaml:",omitempty"`
}

type TemplateSetup struct {
	Kind        string           `yaml:",omitempty"`
	Directories []string         `yaml:",omitempty"`
	Files       []string         `yaml:",omitempty"`
	Extra       []TemplateTarget `yaml:",omitempty"`
}

type Manifest struct {
	Version int
	Kind    string
	Meta    Metadata

	Components []ComponentRef `yaml:",omitempty"`

	Requires []string         `yaml:",omitempty"`
	Provides []string         `yaml:",omitempty"`
	Platform PlatformMetadata `yaml:",omitempty"`

	Lifecycle  Lifecycle     `yaml:",omitempty"`
	Outputs    []Output      `yaml:",omitempty"`
	Parameters []Parameter   `yaml:",omitempty"`
	Templates  TemplateSetup `yaml:",omitempty"`

	Document string `yaml:",omitempty"`
}

type ParametersManifest struct {
	Parameters []Parameter
	Outputs    []Output
}

type ParametersBundle struct {
	Name       string
	Parameters []string
}

type WellKnownParametersManifest struct {
	Parameters []Parameter
	Bundles    []ParametersBundle
}

func (p *Parameter) QName() string {
	return parameterQualifiedName(p.Name, p.Component)
}

func parameterQualifiedName(name, component string) string {
	if component != "" {
		return fmt.Sprintf("%s|%s", name, component)
	}
	return name
}
