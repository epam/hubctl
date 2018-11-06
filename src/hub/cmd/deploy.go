package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hub/config"
	"hub/lifecycle"
	"hub/util"
)

var (
	noLoadState                 bool
	loadFinalState              bool
	offsetComponent             string
	limitComponent              string
	guessComponent              bool
	strictParameters            bool
	compressedState             bool
	gitOutputs                  bool
	gitOutputsStatus            bool
	hubEnvironment              string
	hubStackInstance            string
	hubApplication              string
	hubSaveStackInstanceOutputs bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy hub.yaml.elaborate",
	Short: "Deploy stack",
	Long:  `Deploy stack instance by supplying a fully populated Hub Manifest.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return deploy(args)
	},
}

func setOsEnvForNestedCli(manifests []string, stateManifests []string, componentsBaseDir string) {
	// for nested `hub invoke` and `hub render`
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
		util.Warn("Unable to determine path to Hub CLI executable - `hub invoke / render` are broken: %v", err)
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

func maybeTestVerb(verb string, test bool) string {
	if test {
		return verb + "-test"
	}
	return verb
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

	setOsEnvForNestedCli(manifests, stateManifests, componentsBaseDir)

	request := &lifecycle.Request{
		Verb:                     maybeTestVerb(verb, testVerb),
		ManifestFilenames:        manifests,
		StateFilenames:           stateManifests,
		LoadFinalState:           loadFinalState,
		Components:               components,
		OffsetComponent:          offsetComponent,
		LimitComponent:           limitComponent,
		GuessComponent:           guessComponent,
		StrictParameters:         strictParameters,
		OsEnvironmentMode:        osEnvironmentMode,
		EnvironmentOverrides:     environmentOverrides,
		ComponentsBaseDir:        componentsBaseDir,
		PipeOutputInRealtime:     pipeOutputInRealtime,
		CompressedState:          compressedState,
		GitOutputs:               gitOutputs,
		GitOutputsStatus:         gitOutputsStatus,
		Environment:              hubEnvironment,
		StackInstance:            hubStackInstance,
		Application:              hubApplication,
		SaveStackInstanceOutputs: hubSaveStackInstanceOutputs,
	}

	return request, nil
}

func deploy(args []string) error {
	request, err := lifecycleRequest(args, "deploy")
	if err != nil {
		return err
	}
	lifecycle.Execute(request)
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
	initCommonLifecycleFlags(cmd, verb)
	initCommonApiFlags(cmd)
}

func initCommonLifecycleFlags(cmd *cobra.Command, verb string) {
	cmd.Flags().StringVarP(&componentsBaseDir, "base-dir", "b", "",
		"Path to component sources base directory (default to manifest dir)")
	cmd.Flags().BoolVarP(&pipeOutputInRealtime, "pipe", "", true,
		"Pipe sub-commands output to console in real-time")
	cmd.Flags().BoolVarP(&testVerb, "dry", "y", false,
		fmt.Sprintf("Invoke %[1]s-test verb instead of %[1]s", verb))
	cmd.Flags().BoolVarP(&strictParameters, "strict-parameters", "", true,
		"Put only hub-component.yaml declared parameters into component scope")
	cmd.Flags().StringVarP(&osEnvironmentMode, "os-environment", "", "no-tfvars",
		"OS environment mode for child process, one of: everything, no-tfvars, strict")
	cmd.Flags().BoolVarP(&config.SwitchKubeconfigContext, "switch-kube-context", "", false,
		"Switch current Kubeconfig context to new context. Better use kubectl --context=domain.name instead")
}

func initCommonApiFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&hubEnvironment, "hub-environment", "", "",
		"The Id or Name of Control Plane Environment to obtain deployment parameters from")
	cmd.Flags().StringVarP(&hubStackInstance, "hub-stack-instance", "", "",
		"The Id or Domain of Control Plane Stack Instance to obtain deployment parameters from")
	cmd.Flags().StringVarP(&hubApplication, "hub-application", "", "",
		"The Id or Domain of Control Plane Application to obtain deployment parameters from")
}

func init() {
	initDeployUndeployFlags(deployCmd, "deploy")
	deployCmd.Flags().BoolVarP(&compressedState, "compressed-state", "", true,
		"Write gzip compressed state file")
	deployCmd.Flags().BoolVarP(&gitOutputs, "git-outputs", "", true,
		"Produce hub.components.<component-name>.git.* outputs")
	deployCmd.Flags().BoolVarP(&gitOutputsStatus, "git-outputs-status", "", false,
		"Produce hub.components.<component-name>.git.clean = {clean, dirty} which is expensive to calculate")
	deployCmd.Flags().BoolVarP(&hubSaveStackInstanceOutputs, "hub-save-stack-instance-outputs", "", false,
		"Send Stack Instance outputs and provides to Control Plane (--hub-stack-instance must be set)")
	RootCmd.AddCommand(deployCmd)
}
