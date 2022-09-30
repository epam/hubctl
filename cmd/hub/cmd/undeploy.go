// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/lifecycle"
)

var undeployCmd = &cobra.Command{
	Use:   "undeploy hub.yaml.elaborate",
	Short: "Undeploy stack",
	Long:  `Undeploy stack instance.`,
	Annotations: map[string]string{
		"usage-metering": "tags",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pipe := cmdContextPipe(cmd)
		if pipe != nil {
			defer pipe.Close()
		}
		return undeploy(args, pipe)
	},
}

func undeploy(args []string, pipe io.WriteCloser) error {
	request, err := lifecycleRequest(args, "undeploy")
	if err != nil {
		return err
	}
	lifecycle.Execute(request, pipe)
	return nil
}

func init() {
	initDeployUndeployFlags(undeployCmd, "undeploy")
	undeployCmd.Flags().BoolVarP(&guessComponent, "guess", "", true,
		"Guess component to start undeploy with (useful for failed deployments)")
	RootCmd.AddCommand(undeployCmd)
}
