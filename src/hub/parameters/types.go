package parameters

import (
	"fmt"
)

type LockedParameter struct {
	Component string `yaml:",omitempty"`
	Name      string
	Value     string
	Env       string `yaml:",omitempty"`
}

type RawOutput struct {
	Name  string
	Value string
}

type CapturedOutput struct {
	Component string `yaml:",omitempty" json:"component,omitempty"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	Brief     string `yaml:",omitempty" json:"brief,omitempty"`
	Kind      string `yaml:",omitempty" json:"kind,omitempty"`
}

type LockedParameters map[string]LockedParameter
type RawOutputs map[string]string
type CapturedOutputs map[string]CapturedOutput

type ExpandedOutput struct {
	Name  string
	Value string
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
