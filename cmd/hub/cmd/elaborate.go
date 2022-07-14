// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/compose"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	elaborateOutput                  string
	elaboratePlatformProvides        string
	elaborateUseStateStackParameters bool
)

var elaborateCmd = &cobra.Command{
	Use:   "elaborate hub.yaml [hub-parameters.yaml ...] [-s hub.yaml.state] [-o hub.yaml.elaborate]",
	Short: "Assemble hub.yaml.elaborate",
	Long: `Assemble a complete Stack or Application deployment manifest by joining stack and components manifests.
Parameters are injected from parameters manifest(s) and optionally are read from state file.
The resulted hub.yaml.elaborate can be used with deploy command.`,
	Annotations: map[string]string{
		"usage-metering": "tags",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pipe := cmdContextPipe(cmd)
		if pipe != nil {
			defer pipe.Close()
		}
		return elaborate(args, pipe)
	},
}

func elaborate(args []string, pipe io.WriteCloser) error {
	if len(args) < 1 {
		return errors.New("Elaborate command has one or more arguments - path to Stack Manifest file and optionally to parameters file(s)")
	}

	manifest := args[0]
	parameters := []string{}
	if len(args) > 1 {
		parameters = args[1:]
	}
	elaborateManifests := util.SplitPaths(elaborateOutput)
	stateManifests := util.SplitPaths(stateManifestExplicit)
	compose.Elaborate(manifest, parameters, environmentOverrides, elaboratePlatformProvides,
		stateManifests, elaborateUseStateStackParameters, elaborateManifests, componentsBaseDir,
		pipe)

	return nil
}

func init() {
	elaborateCmd.Flags().StringVarP(&environmentOverrides, "environment", "e", "",
		"Set Hub environment variables: -e 'NAME=demo,INSTANCE=r4.large,...'")
	elaborateCmd.Flags().StringVarP(&elaboratePlatformProvides, "platform-provides", "p", "",
		"Set Platform stack provides: -p tiller,etcd,...")
	elaborateCmd.Flags().StringVarP(&elaborateOutput, "output", "o", "hub.yaml.elaborate",
		"Set output filename")
	elaborateCmd.Flags().StringVarP(&componentsBaseDir, "baseDir", "b", "",
		"Path to component sources base directory (default to manifest dir)")
	elaborateCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		"Path to state file(s) to load Platform stack outputs as input parameters, for example hub.yaml.state,s3://bucket/hub.yaml.state")
	elaborateCmd.Flags().BoolVarP(&elaborateUseStateStackParameters, "state-stack-parameters", "", true,
		"Also use stack parameters (from state) to load input parameters, otherwise only stack outputs are used")
	RootCmd.AddCommand(elaborateCmd)
}
