// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/api"
)

var stackCmd = &cobra.Command{
	Use:   "stack <get> ...",
	Short: "List Base Stacks",
}

var stackGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Show a list of base stacks or details about the base stack",
	Long: `Show a list of all base stacks or details about
the particular base stack (specify Id)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return stack(args)
	},
}

func stack(args []string) error {
	if len(args) > 1 {
		return errors.New("Stack command has one optional argument - name of the base stack")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.BaseStacks(selector, jsonFormat)

	return nil
}

func init() {
	stackGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	stackCmd.AddCommand(stackGetCmd)
	apiCmd.AddCommand(stackCmd)
}
