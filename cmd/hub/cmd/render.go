// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/lifecycle"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	templateKind         string
	additionalParameters string
)

var renderCmd = &cobra.Command{
	Use:   "render <template glob> ... [-a 'additional.parameter1=value,...']",
	Short: "Render component templates",
	Long:  `Render component templates with additional parameters during lifecycle operation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return render(args)
	},
}

func render(args []string) error {
	if len(args) == 0 {
		return errors.New("Render command has one or more arguments - templates globs/paths")
	}

	if elaborateManifest == "" {
		elaborateManifest = os.Getenv(envVarNameElaborate)
		if elaborateManifest == "" && config.Debug {
			log.Printf("%s environment variable should be set to hub.yaml.elaborate filename(s); using parameters locked in state file instead",
				envVarNameElaborate)
		}
	}
	if stateManifestExplicit == "" {
		stateManifestExplicit = os.Getenv(envVarNameState)
		if stateManifestExplicit == "" {
			return fmt.Errorf("%s environment variable must be set to hub.yaml.state filename(s)", envVarNameState)
		}
	}
	manifests := util.SplitPaths(elaborateManifest)
	stateManifests := util.SplitPaths(stateManifestExplicit)
	if componentName == "" {
		componentName = os.Getenv(lifecycle.HubEnvVarNameComponentName)
		if componentName == "" && config.Debug {
			log.Printf("%s environment variable should be set to component name; using stack-level parameters and outputs",
				lifecycle.HubEnvVarNameComponentName)
		}
	}

	config.AggWarnings = false

	lifecycle.Render(manifests, stateManifests, componentName,
		templateKind, additionalParameters, args)

	return nil
}

func init() {
	renderCmd.Flags().StringVarP(&elaborateManifest, "elaborate", "m", "",
		fmt.Sprintf("Path to hub.yaml.elaborate manifest file (default from %s environment variable)", envVarNameElaborate))
	renderCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		fmt.Sprintf("Path to state files (default from %s environment variable)", envVarNameState))
	renderCmd.Flags().StringVarP(&componentName, "component", "c", "",
		fmt.Sprintf("Component name to load state at (default from %s environment variable)", lifecycle.HubEnvVarNameComponentName))
	renderCmd.Flags().StringVarP(&templateKind, "kind", "k", "curly",
		"`curly`, mustache, go")
	renderCmd.Flags().StringVarP(&additionalParameters, "additional-parameters", "a", "",
		"Set additional parameters: -a 'component.password=qwerty,...'")
	RootCmd.AddCommand(renderCmd)
}
