// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/initialize"
)

var initCmd = &cobra.Command{
	Use:   "init <stack | component> [-f] [dir]",
	Short: "Init stack or component manifest",
	Long:  `Create stack or component manifest with initial values provided.`,
}

var initStackCmd = &cobra.Command{
	Use:   "stack [dir]",
	Short: "Init stack manifest",
	Long:  `Create stack manifest with initial values provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeDir(args, true)
	},
}

var initComponentCmd = &cobra.Command{
	Use:   "component [dir]",
	Short: "Init component manifest",
	Long:  `Create component manifest with initial values provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeDir(args, false)
	},
}

func initializeDir(args []string, stack bool) error {
	if len(args) != 1 && len(args) != 0 {
		return errors.New("Init command has only one optional argument - directory for Manifest file (default to `.`)")
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}

	if stack {
		initialize.InitStack(dir)
	} else {
		initialize.InitComponent(dir)
	}

	return nil
}

func init() {
	initCmd.AddCommand(initStackCmd)
	initCmd.AddCommand(initComponentCmd)
	RootCmd.AddCommand(initCmd)
}
