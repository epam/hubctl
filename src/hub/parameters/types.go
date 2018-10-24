package parameters

import (
	"fmt"

	"hub/manifest"
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
}

type LockedParameters map[string]LockedParameter
type RawOutputs map[string]string
type CapturedOutputs map[string]CapturedOutput

type ExpandedOutput struct {
	Name  string
	Value string
	Brief string
	Kind  string
}

func (p *LockedParameter) QName() string {
	return lockedParameterQualifiedName(p)
}

func (o *CapturedOutput) QName() string {
	return capturedOutputQualifiedName(o)
}

func manifestParameterQualifiedName(parameter *manifest.Parameter) string {
	return parameterQualifiedName(parameter.Name, parameter.Component)
}

func lockedParameterQualifiedName(parameter *LockedParameter) string {
	return parameterQualifiedName(parameter.Name, parameter.Component)
}

func capturedOutputQualifiedName(output *CapturedOutput) string {
	return OutputQualifiedName(output.Name, output.Component)
}

func parameterQualifiedName(name string, component string) string {
	if component != "" {
		return fmt.Sprintf("%s|%s", name, component)
	}
	return name
}

func OutputQualifiedName(name string, component string) string {
	if component != "" {
		return fmt.Sprintf("%s:%s", component, name)
	}
	return name
}
