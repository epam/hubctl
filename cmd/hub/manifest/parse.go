// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package manifest

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/bindata"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/storage"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func ParseManifest(manifestFilenames []string) (*Manifest, []Manifest, string, error) {
	yamlBytes, manifestFilename, err := storage.CheckAndRead(manifestFilenames, "manifest")
	if err != nil {
		return nil, nil, manifestFilename, fmt.Errorf("Unable to read %v manifest: %v", manifestFilenames, err)
	}

	yamlDocuments := bytes.Split(yamlBytes, []byte("\n---\n"))

	var manifests []Manifest
	for i, yamlDocument := range yamlDocuments {
		if len(yamlDocument) == 0 {
			continue
		}
		validateManifest(fmt.Sprintf("%s[%d]", manifestFilename, i), yamlDocument)
		var manifest Manifest
		err = yaml.Unmarshal(yamlDocument, &manifest)
		if err != nil {
			return nil, nil, manifestFilename, fmt.Errorf("Unable to parse %s (doc %d/%d): %v",
				manifestFilename, i+1, len(yamlDocuments), err)
		}
		manifest.Document = string(yamlDocument)
		manifests = append(manifests, manifest)
	}
	if len(manifests) == 0 {
		return nil, nil, manifestFilename, fmt.Errorf("No YAML documents in %s", manifestFilename)
	} else if len(manifests) > 1 {
		return &manifests[0], manifests[1:], manifestFilename, nil
	} else {
		return &manifests[0], nil, manifestFilename, nil
	}
}

func ParseParametersManifest(manifestFilenames []string) (*ParametersManifest, string, error) {
	yamlBytes, manifestFilename, err := storage.CheckAndRead(manifestFilenames, "parameters")
	if err != nil {
		return nil, manifestFilename, fmt.Errorf("Unable to read %v: %v", manifestFilenames, err)
	}

	yamlDocuments := bytes.Split(yamlBytes, []byte("\n---\n"))

	for i, yamlDocument := range yamlDocuments {
		if len(yamlDocument) == 0 {
			continue
		}
		validateManifest(manifestFilename, yamlDocument)
		var manifest ParametersManifest
		err = yaml.Unmarshal(yamlDocument, &manifest)
		if err != nil {
			return nil, manifestFilename, fmt.Errorf("Unable to parse %s: %v", manifestFilename, err)
		}
		if len(yamlDocuments) > i+1 {
			util.Warn("Parameters manifest `%s` contains more than one YAML document, only first is used",
				manifestFilename)
		}
		if len(manifest.Parameters) == 0 && config.Verbose {
			log.Printf("Parameters manifest `%s` contains no parameters",
				manifestFilename)
		}
		return &manifest, manifestFilename, nil
	}

	return nil, manifestFilename, fmt.Errorf("No YAML documents found in %s", manifestFilename)
}

func GetWellKnownParametersManifest() (*WellKnownParametersManifest, error) {
	yamlBytes, err := bindata.Asset("meta/hub-well-known-parameters.yaml")
	if err != nil {
		return nil, fmt.Errorf("No well-known parameters embedded: %v", err)
	}
	yamlDocuments := bytes.Split(yamlBytes, []byte("\n---\n"))
	for i, yamlDocument := range yamlDocuments {
		if len(yamlDocument) == 0 {
			continue
		}
		var manifest WellKnownParametersManifest
		err = yaml.Unmarshal(yamlDocument, &manifest)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse well-known parameters: %v", err)
		}
		if len(yamlDocuments) > i+1 {
			util.Warn("Embedded well-known parameters manifest contains more than one YAML document, only first is used")
		}
		return &manifest, nil
	}

	return nil, fmt.Errorf("No YAML documents found in embedded well-known parameters manifest")
}

func ParseComponentsManifests(components []ComponentRef, stackBaseDir string, componentsBaseDir string) ([]Manifest, error) {
	manifests := make([]Manifest, 0, len(components))
	for _, component := range components {
		dir := ComponentSourceDirFromRef(&component, stackBaseDir, componentsBaseDir)
		filename := filepath.Join(dir, "hub-component.yaml")
		manifest, rest, _, err := ParseManifest([]string{filename})
		if err != nil {
			return nil, fmt.Errorf("Unable to parse component `%s` manifest: %v",
				ComponentQualifiedNameFromRef(&component), err)
		}
		if len(rest) > 0 {
			util.Warn("Component `%s` manifest `%s` contains more than one YAML document, only first is used",
				ComponentQualifiedNameFromRef(&component), filename)
		}
		if len(manifest.Parameters) == 0 && config.Verbose {
			log.Printf("Component manifest `%s` contains no parameters",
				filename)
		}
		if manifest.Meta.Origin == "" {
			manifest.Meta.Origin = manifest.Meta.Name
		}
		if manifest.Meta.Kind == "" {
			manifest.Meta.Kind = manifest.Meta.Name
		}
		manifest.Meta.Name = component.Name
		manifests = append(manifests, *manifest)
	}

	return manifests, nil
}

func ParseComponentsManifestsWithExclusion(components []ComponentRef, excludedComponents []string,
	stackBaseDir string, componentsBaseDir string) ([]Manifest, error) {

	if config.Debug && len(excludedComponents) > 0 {
		log.Printf("Components excluded from parsing (in parent stack):\n\t%s",
			strings.Join(excludedComponents, ", "))
	}

	filtered := make([]ComponentRef, 0, len(components))
	for _, ref := range components {
		fqName := ComponentQualifiedNameFromRef(&ref)
		found := false
		for _, excluded := range excludedComponents {
			if fqName == excluded {
				found = true
				break
			}
		}
		if !found {
			filtered = append(filtered, ref)
		}
	}

	return ParseComponentsManifests(filtered, stackBaseDir, componentsBaseDir)
}

func GenerateLifecycleOrder(manifest *Manifest) ([]string, error) {
	componentsOrder := manifest.Lifecycle.Order
	notDefinedComponents := make([]string, 0)

	if len(componentsOrder) == 0 {
		componentsMap := make(map[string]ComponentRef)
		for _, ref := range manifest.Components {
			componentsMap[ref.Name] = ref
		}

		componentsOrder = make([]string, 0)
		for _, ref := range manifest.Components {
			if util.Contains(componentsOrder, ref.Name) {
				continue
			}
			dependencies, notDefinedDeps, err := resolveComponentDependencies(ref, componentsMap, "")
			if err != nil {
				return nil, err
			}
			for _, dependency := range dependencies {
				if !util.Contains(componentsOrder, dependency) {
					componentsOrder = append(componentsOrder, dependency)
				}
			}
			notDefinedComponents = appendUniq(notDefinedComponents, notDefinedDeps)
			componentsOrder = append(componentsOrder, ref.Name)
		}
	}

	if len(notDefinedComponents) > 0 {
		return nil, fmt.Errorf("These components are not defined but used as dependencies: %s", strings.Join(notDefinedComponents, ", "))
	}

	return componentsOrder, nil
}

func resolveComponentDependencies(ref ComponentRef, componentsMap map[string]ComponentRef, prevComponentName string) ([]string, []string, error) {
	componentDependensies := ref.Depends
	if len(componentDependensies) == 0 {
		return componentDependensies, nil, nil
	}

	if util.Contains(componentDependensies, ref.Name) || util.Contains(componentDependensies, prevComponentName) {
		return nil, nil, fmt.Errorf("Components dependency cycle detected in definition of \"%s\" component", ref.Name)
	}

	result := make([]string, 0)
	notDefined := make([]string, 0)
	for _, dependencyName := range componentDependensies {
		if util.Contains(result, dependencyName) {
			continue
		}
		dependencyRef, exists := componentsMap[dependencyName]
		if exists {
			dependenciesChain, notDefinedChain, err := resolveComponentDependencies(dependencyRef, componentsMap, ref.Name)
			if err != nil {
				return nil, nil, err
			}
			result = appendUniq(result, dependenciesChain)
			notDefined = appendUniq(notDefined, notDefinedChain)
			result = append(result, dependencyName)
		} else {
			if !util.Contains(notDefined, dependencyName) {
				notDefined = append(notDefined, dependencyName)
			}
		}
	}

	return result, notDefined, nil
}

func appendUniq(result []string, items []string) []string {
	for _, item := range items {
		if !util.Contains(result, item) {
			result = append(result, item)
		}
	}

	return result
}
