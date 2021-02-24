package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/lifecycle"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	noLoadState                   bool
	loadFinalState                bool
	enabledClouds                 string
	offsetComponent               string
	limitComponent                string
	guessComponent                bool
	compressedState               bool
	gitOutputs                    bool
	gitOutputsStatus              bool
	hubEnvironment                string
	hubStackInstance              string
	hubApplication                string
	hubSaveStackInstanceOutputs   bool
	hubSyncStackInstance          bool
	hubSyncSkipParametersAndOplog bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy hub.yaml.elaborate",
	Short: "Deploy stack",
	Long:  `Deploy stack instance by supplying a fully populated Hub Manifest.`,
	Annotations: map[string]string{
		"usage-metering": "tags",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pipe := cmdContextPipe(cmd)
		if pipe != nil {
			defer pipe.Close()
		}
		return deploy(args, pipe)
	},
}

func setOsEnvForNestedCli(manifests []string, stateManifests []string, componentsBaseDir string) {
	// for nested `hub invoke`, `render`, and `util otp`
	if bin, err := os.Executable(); err == nil {
		os.Setenv(envVarNameHubCli, bin)
		// TODO a local state that is not up-to-date, but remote is?
		os.Setenv(envVarNameElaborate, statePreferLocalFile(manifests))
		os.Setenv(envVarNameState, statePreferLocalFile(stateManifests))
		os.Setenv(envVarNameAwsRegion, config.AwsRegion)
		if componentsBaseDir != "" {
			os.Setenv(envVarNameComponentsBaseDir, componentsBaseDir)
		}
	} else {
		util.Warn("Unable to determine path to Hub CLI executable - `hub invoke / render / util otp` are broken: %v", err)
	}
}

func statePreferLocalFile(filenames []string) string {
	if len(filenames) == 0 {
		return ""
	}
	for _, name := range filenames {
		if !strings.Contains(name, "://") {
			if _, err := os.Stat(name); err == nil {
				return util.MustAbs(name)
			}
		}
	}
	for _, name := range filenames {
		if strings.Contains(name, "://") {
			return name
		}
	}
	return filenames[0]
}

func lifecycleRequest(args []string, verb string) (*lifecycle.Request, error) {
	if len(args) != 1 {
		mirrorArgs := ""
		if len(args) > 0 {
			mirrorArgs = fmt.Sprintf(", %v were supplied", args)
		}
		return nil, fmt.Errorf("`%s` command has one argument - path to Manifest file%s", verb, mirrorArgs)
	}

	if noLoadState {
		stateManifest = ""
	}

	if componentName != "" && offsetComponent != "" {
		return nil, errors.New("At most one of -c / --components or -o / --offset must be specified")
	}
	if (componentName != "" || offsetComponent != "") && stateManifest == "" && !config.Force {
		return nil, errors.New("State file (-s) must be specified when component (-c or -o) is specified")
	}

	manifests := util.SplitPaths(args[0])
	stateManifests := util.SplitPaths(stateManifest)
	components := util.SplitPaths(componentName)
	clouds := util.SplitPaths(strings.ToLower(enabledClouds))
	if !util.ContainsAll(supportedClouds, clouds) {
		return nil, fmt.Errorf("Unsupported cloud specified (--clouds): %s; supported clouds are: %s",
			strings.Join(clouds, ", "), strings.Join(supportedClouds, ", "))
	}

	setOsEnvForNestedCli(manifests, stateManifests, componentsBaseDir)

	// TODO remove compat
	if hubSaveStackInstanceOutputs {
		hubSyncStackInstance = true
		hubSyncSkipParametersAndOplog = true
	}

	request := &lifecycle.Request{
		Verb:                       verb,
		DryRun:                     dryRun,
		ManifestFilenames:          manifests,
		StateFilenames:             stateManifests,
		LoadFinalState:             loadFinalState,
		EnabledClouds:              clouds,
		Components:                 components,
		OffsetComponent:            offsetComponent,
		LimitComponent:             limitComponent,
		GuessComponent:             guessComponent,
		OsEnvironmentMode:          osEnvironmentMode,
		EnvironmentOverrides:       environmentOverrides,
		ComponentsBaseDir:          componentsBaseDir,
		GitOutputs:                 gitOutputs,
		GitOutputsStatus:           gitOutputsStatus,
		Environment:                hubEnvironment,
		StackInstance:              hubStackInstance,
		Application:                hubApplication,
		SyncStackInstance:          hubSyncStackInstance,
		SyncSkipParametersAndOplog: hubSyncSkipParametersAndOplog,
	}

	return request, nil
}

func deploy(args []string, pipe io.WriteCloser) error {
	request, err := lifecycleRequest(args, "deploy")
	if err != nil {
		return err
	}
	lifecycle.Execute(request, pipe)
	return nil
}

func initDeployUndeployFlags(cmd *cobra.Command, verb string) {
	cmd.Flags().StringVarP(&stateManifest, "state", "s", "hub.yaml.state",
		"Path to state file(s), for example hub.yaml.state,s3://bucket/hub.yaml.state")
	cmd.Flags().BoolVarP(&noLoadState, "no-state", "n", false,
		"Skip state file load")
	cmd.Flags().BoolVarP(&loadFinalState, "load-global-state", "g", false,
		"Load global (final) state instead of the specific component (applies to -c, -o)")
	cmd.Flags().StringVarP(&componentName, "components", "c", "",
		fmt.Sprintf("A list of components to %s (separated by comma; state file must exist)", verb))
	cmd.Flags().StringVarP(&offsetComponent, "offset", "o", "",
		fmt.Sprintf("Component to start %s with (state file must exist)", verb))
	cmd.Flags().StringVarP(&limitComponent, "limit", "l", "",
		fmt.Sprintf("Component to stop %s at", verb))
	cmd.Flags().StringVarP(&environmentOverrides, "environment", "e", "",
		"Set environment overrides: -e 'NAME=demo,INSTANCE=r4.large,...'")
	cmd.Flags().BoolVarP(&hubSyncStackInstance, "hub-sync", "", false,
		"Sync Stack Instance state to SuperHub (--hub-stack-instance must be set)")
	cmd.Flags().BoolVarP(&hubSyncSkipParametersAndOplog, "hub-sync-skip-parameters-and-oplog", "", false,
		"Sync skip syncing Stack Instance parameters and operation log")
	initCommonLifecycleFlags(cmd, verb)
	initCommonApiFlags(cmd)
}

func initCommonLifecycleFlags(cmd *cobra.Command, verb string) {
	cmd.Flags().StringVarP(&componentsBaseDir, "base-dir", "b", "",
		"Path to component sources base directory (default to manifest dir)")
	cmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		fmt.Sprintf("Invoke %[1]s-test verb instead of %[1]s", verb))
	cmd.Flags().StringVarP(&osEnvironmentMode, "os-environment", "", "no-tfvars",
		"OS environment mode for child process, one of: everything, no-tfvars, strict")
	cmd.Flags().BoolVarP(&config.SwitchKubeconfigContext, "switch-kube-context", "", false,
		"Switch current Kubeconfig context to new context. Use kubectl --context=domain.name instead")
	cmd.Flags().StringVarP(&enabledClouds, "clouds", "", "",
		"A list of enabled clouds: \"aws,azure,gcp\" (default to autodetect from environment)")
}

func initCommonApiFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&hubEnvironment, "hub-environment", "", "",
		"The Id or Name of SuperHub Environment to obtain deployment parameters from")
	cmd.Flags().StringVarP(&hubStackInstance, "hub-stack-instance", "", "",
		"The Id or Domain of SuperHub Stack Instance to obtain deployment parameters from")
	cmd.Flags().StringVarP(&hubApplication, "hub-application", "", "",
		"The Id or Domain of SuperHub Application to obtain deployment parameters from")
}

func init() {
	initDeployUndeployFlags(deployCmd, "deploy")
	deployCmd.Flags().BoolVarP(&gitOutputs, "git-outputs", "", true,
		"Produce hub.components.<component-name>.git.* outputs")
	deployCmd.Flags().BoolVarP(&gitOutputsStatus, "git-outputs-status", "", false,
		"Produce hub.components.<component-name>.git.clean = {clean, dirty} which is expensive to calculate")
	deployCmd.Flags().BoolVarP(&hubSaveStackInstanceOutputs, "hub-save-stack-instance-outputs", "", false,
		"(deprecated) Send Stack Instance outputs and provides to SuperHub (--hub-stack-instance must be set)")
	RootCmd.AddCommand(deployCmd)
}
