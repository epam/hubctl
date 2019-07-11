package manifest

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"hub/bindata"
	"hub/config"
	"hub/storage"
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
		var manifest ParametersManifest
		err = yaml.Unmarshal(yamlDocument, &manifest)
		if err != nil {
			return nil, manifestFilename, fmt.Errorf("Unable to parse %s: %v", manifestFilename, err)
		}
		if len(yamlDocuments) > i+1 {
			log.Printf("Parameters manifest `%s` contains more than one YAML document, only first is used",
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
			log.Print("Embedded well-known parameters manifest contains more than one YAML document, only first is used")
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
			log.Printf("Component `%s` manifest `%s` contains more than one YAML document, only first is used",
				ComponentQualifiedNameFromRef(&component), filename)
		}
		if component.Name != manifest.Meta.Name {
			manifest.Meta.Name = component.Name
		}
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
