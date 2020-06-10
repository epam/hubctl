package state

import (
	"fmt"
	"log"

	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/storage"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func MustParseStateFiles(stateManifests []string) *StateManifest {
	stateFiles, errs := storage.Check(stateManifests, "state")
	if len(errs) > 0 {
		log.Fatalf("Unable to check state files: %s", util.Errors2(errs...))
	}
	state, err := ParseState(stateFiles)
	if err != nil {
		log.Fatalf("Unable to load state: %v", err)
	}
	return state
}

func ParseState(files *storage.Files) (*StateManifest, error) {
	yamlDocument, stateFilename, err := storage.Read(files)
	if err != nil {
		return nil, err
	}

	var state StateManifest
	err = yaml.Unmarshal(yamlDocument, &state)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse `%s`: %v", stateFilename, err)
	}

	if config.Trace {
		printState(&state)
	}

	if state.Kind != "state" {
		return nil, fmt.Errorf("State file kind = `%s` but it must be `state`", state.Kind)
	}

	return &state, nil
}

func MustParseBackupBundles(backupBundles []string) *BackupManifest {
	bundleFiles, errs := storage.Check(backupBundles, "backup bundle")
	if len(errs) > 0 {
		log.Fatalf("Unable to check backup bundle files: %s", util.Errors2(errs...))
	}
	bundle, err := ParseBackupBundle(bundleFiles)
	if err != nil {
		log.Fatalf("Unable to load backup bundle: %v", err)
	}
	return bundle
}

func ParseBackupBundle(files *storage.Files) (*BackupManifest, error) {
	yamlDocument, bundleFilename, err := storage.Read(files)
	if err != nil {
		return nil, err
	}

	var bundle BackupManifest
	err = yaml.Unmarshal(yamlDocument, &bundle)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse `%s`: %v", bundleFilename, err)
	}

	if bundle.Kind != "backup" {
		return nil, fmt.Errorf("Backup bundle file kind = `%s` but it must be `backup`", bundle.Kind)
	}

	bundle.Source = bundleFilename
	return &bundle, nil
}
