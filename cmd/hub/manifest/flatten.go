// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package manifest

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

func FlattenParameters(parameters []Parameter, tag string) []Parameter {
	flattened := flattenParametersWithPrefix("", "", "", "", parameters)
	if config.Debug {
		if len(flattened) > 0 {
			log.Printf("Parameters flattened (%s):", tag)
			printParameters(flattened)
		} else {
			log.Printf("No parameters found in %s", tag)
		}
	}
	return flattened
}

func flattenParametersWithPrefix(prefix string, kind string, empty string, component string, parameters []Parameter) []Parameter {
	if len(parameters) == 0 {
		return []Parameter{}
	}
	flattened := make([]Parameter, 0)
	for _, parameter := range parameters {
		if parameter.Name == "" {
			util.Warn("Component `%s` parameter name is empty at prefix `%s`, value `%s`",
				component, prefix, parameter.Value)
		}
		// let pretend only leafs are important
		if len(parameter.Parameters) > 0 {
			nested := flattenParametersWithPrefix(
				fmt.Sprintf("%s%s.", prefix, parameter.Name),
				mergeField(kind, parameter.Kind), mergeField(empty, parameter.Empty), mergeField(component, parameter.Component),
				parameter.Parameters)
			flattened = append(flattened, nested...)
		} else {
			parameter.Name = fmt.Sprintf("%s%s", prefix, parameter.Name)
			if parameter.Kind == "" && kind != "" {
				parameter.Kind = kind
			}
			if parameter.Empty == "" && empty != "" {
				parameter.Empty = empty
			}
			if parameter.Component == "" && component != "" {
				parameter.Component = component
			}
			flattened = append(flattened, parameter)
		}
	}
	return flattened
}

func mergeField(base string, over string) string {
	out := base
	if over != "" {
		out = over
	}
	return out
}

func ComponentRefByName(components []ComponentRef, componentName string) *ComponentRef {
	for i, component := range components {
		name := ComponentQualifiedNameFromRef(&component)
		if name == componentName {
			return &components[i]
		}
	}
	return nil
}

func ComponentManifestByRef(componentsManifests []Manifest, component *ComponentRef) *Manifest {
	name := ComponentQualifiedNameFromRef(component)
	for _, componentsManifest := range componentsManifests {
		if name == ComponentQualifiedNameFromMeta(&componentsManifest.Meta) {
			return &componentsManifest
		}
	}
	return nil
}

func ComponentsNamesFromManifests(manifests []Manifest) []string {
	res := make([]string, len(manifests))
	for i, v := range manifests {
		res[i] = ComponentQualifiedNameFromMeta(&v.Meta)
	}
	return res
}

func ComponentsNamesFromRefs(components []ComponentRef) []string {
	res := make([]string, len(components))
	for i, v := range components {
		res[i] = ComponentQualifiedNameFromRef(&v)
	}
	return res
}

func ComponentQualifiedNameFromRef(ref *ComponentRef) string {
	return ref.Name
}

func ComponentQualifiedNameFromMeta(meta *Metadata) string {
	return meta.Name
}

func ComponentSourceDirNameFromRef(component *ComponentRef) string {
	return component.Name
}

func ComponentSourceDirFromRef(component *ComponentRef, stackBaseDir, componentsBaseDir string) string {
	dir := ""
	source := component.Source
	if source.Dir != "" {
		if !filepath.IsAbs(source.Dir) {
			dir = filepath.Join(stackBaseDir, source.Dir)
		} else {
			dir = source.Dir
		}
	}
	if dir == "" {
		if source.Git.LocalDir != "" {
			dir = filepath.Join(source.Git.LocalDir, source.Git.SubDir)
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(componentsBaseDir, dir)
			}
		} else if source.Git.Remote != "" {
			dir = filepath.Join(componentsBaseDir, ComponentSourceDirNameFromRef(component), source.Git.SubDir)
		}
	}
	if dir == "" {
		log.Fatalf("No source directory set for component `%s`", ComponentQualifiedNameFromRef(component))
	}

	if config.Trace {
		log.Printf("Component `%s` source dir: `%s`", ComponentQualifiedNameFromRef(component), dir)
	}

	return dir
}

func MakeParameters(names []string) []Parameter {
	params := make([]Parameter, 0, len(names))
	for _, name := range names {
		params = append(params, Parameter{Name: name})
	}
	return params
}
