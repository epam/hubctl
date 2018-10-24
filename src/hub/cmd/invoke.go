package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"hub/lifecycle"
	"hub/util"
)

var invokeCmd = &cobra.Command{
	Use:   "invoke <component> <verb> [-e 'ADDITIONAL_ENV_VAR1=value,...']",
	Short: "Invoke component's verb (from another component)",
	Long:  `Invoke stack component's verb from other component during lifecycle operation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return invoke(args)
	},
}

func invoke(args []string) error {
	if len(args) != 2 {
		return errors.New("Invoke command has two mandatory argument - component name and verb")
	}

	component := args[0]
	verb := args[1]

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
	if componentsBaseDir == "" {
		componentsBaseDir = os.Getenv(envVarNameComponentsBaseDir)
	}

	setOsEnvForNestedCli(manifests, stateManifests, componentsBaseDir)

	request := &lifecycle.Request{
		Component:            component,
		Verb:                 verb,
		ManifestFilenames:    manifests,
		StateFilenames:       stateManifests,
		OsEnvironmentMode:    osEnvironmentMode,
		EnvironmentOverrides: environmentOverrides,
		ComponentsBaseDir:    componentsBaseDir,
		PipeOutputInRealtime: pipeOutputInRealtime,
	}
	lifecycle.Invoke(request)

	return nil
}

func init() {
	invokeCmd.Flags().StringVarP(&elaborateManifest, "elaborate", "m", "",
		fmt.Sprintf("Path to hub.yaml.elaborate manifest file (default from %s environment variable)", envVarNameElaborate))
	invokeCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		fmt.Sprintf("Path to state files (default from %s environment variable)", envVarNameState))
	invokeCmd.Flags().StringVarP(&componentsBaseDir, "base-dir", "b", "",
		fmt.Sprintf("Path to component sources base directory (default from %s environment variable, then manifest dir)", envVarNameComponentsBaseDir))
	invokeCmd.Flags().StringVarP(&osEnvironmentMode, "os-environment", "", "no-tfvars",
		"OS environment mode for child process, one of: everything, no-tfvars, strict")
	invokeCmd.Flags().StringVarP(&environmentOverrides, "environment", "e", "",
		"Set additional environment variables: -e 'PORT=5000,...'")
	invokeCmd.Flags().BoolVarP(&pipeOutputInRealtime, "pipe", "p", true,
		"Pipe sub-commands output to console in real-time")
	RootCmd.AddCommand(invokeCmd)
}
