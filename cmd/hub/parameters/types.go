package parameters

import (
	"fmt"
)

type LockedParameter struct {
	Component string `yaml:",omitempty"`
	Name      string
	Value     interface{}
	Env       string `yaml:",omitempty"`
}

type RawOutput struct {
	Name  string
	Value string
}

type CapturedOutput struct {
	Component       string      `yaml:",omitempty" json:"component,omitempty"`
	ComponentOrigin string      `yaml:"componentOrigin,omitempty" json:"-"`
	ComponentKind   string      `yaml:"componentKind,omitempty" json:"-"`
	Name            string      `json:"name"`
	Value           interface{} `json:"value"`
	Brief           string      `yaml:",omitempty" json:"brief,omitempty"`
	Kind            string      `yaml:",omitempty" json:"kind,omitempty"`
}

type LockedParameters map[string]LockedParameter
type RawOutputs map[string]string
type CapturedOutputs map[string]CapturedOutput

type ExpandedOutput struct {
	Name  string
	Value interface{}
	Brief string `yaml:",omitempty"`
	Kind  string `yaml:",omitempty"`
}

func (p *LockedParameter) QName() string {
	return lockedParameterQualifiedName(p)
}

func (o *CapturedOutput) QName() string {
	return capturedOutputQualifiedName(o)
}

func lockedParameterQualifiedName(parameter *LockedParameter) string {
	return parameterQualifiedName(parameter.Name, parameter.Component)
}

func capturedOutputQualifiedName(output *CapturedOutput) string {
	return OutputQualifiedName(output.Name, output.Component)
}

func parameterQualifiedName(name, component string) string {
	if component != "" {
		return fmt.Sprintf("%s|%s", name, component)
	}
	return name
}

func OutputQualifiedName(name, component string) string {
	if component != "" {
		return fmt.Sprintf("%s:%s", component, name)
	}
	return name
}
