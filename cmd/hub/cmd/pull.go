// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/git"
)

var (
	reset              bool
	recurse            bool
	subtree            bool
	optimizeGitRemotes bool
)

var pullCmd = &cobra.Command{
	Use:   "pull hub.yaml [-b <base directory>] [-f] [-r]",
	Short: "Pull stack sources",
	Long:  `Clone or update stack and component sources from Git.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pull(args)
	},
}

func pull(args []string) error {
	if len(args) != 1 {
		return errors.New("Pull command has only one argument - path to Stack Manifest file")
	}

	manifest := args[0]
	git.PullManifest(manifest, componentsBaseDir, reset, recurse, optimizeGitRemotes, subtree)

	return nil
}

func init() {
	pullCmd.Flags().StringVarP(&componentsBaseDir, "baseDir", "b", "",
		"Path to base directory to clone sources into (default to manifest dir)")
	pullCmd.Flags().BoolVarP(&optimizeGitRemotes, "optimize-git-remotes", "", true,
		"Optimize Git remote with local clone (same remote repository is encountered more than once)")
	pullCmd.Flags().BoolVarP(&reset, "reset", "r", false,
		"Stash and reset Git tree prior to update")
	pullCmd.Flags().BoolVarP(&recurse, "recurse", "", true,
		"Recurse into `fromStack`")
	pullCmd.Flags().BoolVarP(&subtree, "subtree", "s", false,
		"Pull components as Git subtrees")
	// RootCmd.AddCommand(pullCmd)
}
