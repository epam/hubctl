package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hub/lifecycle"
	"hub/util"
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
		if elaborateManifest == "" {
			return fmt.Errorf("%s environment variable must be set to hub.yaml.elaborate filename(s)", envVarNameElaborate)
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
		if componentName == "" {
			return fmt.Errorf("%s environment variable must be set to component name", lifecycle.HubEnvVarNameComponentName)
		}
	}

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
		"`curly` or mustache")
	renderCmd.Flags().StringVarP(&additionalParameters, "additional-parameters", "a", "",
		"Set additional parameters: -a 'component.password=qwerty,...'")
	RootCmd.AddCommand(renderCmd)
}
