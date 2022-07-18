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

	"github.com/agilestacks/hub/cmd/hub/api"
)

var apiBackupCmd = &cobra.Command{
	Use:   "backup <get | delete> ...",
	Short: "List and manage Stack Instance backups",
}

var apiBackupGetCmd = &cobra.Command{
	Use:   "get [id | name]",
	Short: "Show a list of Backups or details about the Backup",
	Long: `Show a list of all user accessible Backups or details about
the particular Backup (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return apiBackup(args)
	},
}

var apiBackupDeleteCmd = &cobra.Command{
	Use:   "delete <id | name>",
	Short: "Delete Backup by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteApiBackup(args)
	},
}

func apiBackup(args []string) error {
	if len(args) > 1 {
		return errors.New("Backup command has one optional argument - id or name of the backup")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Backups(selector, showLogs, jsonFormat)

	return nil
}

func deleteApiBackup(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Backup command has one mandatory argument - id or name of the backup")
	}

	api.DeleteBackup(args[0])

	return nil
}

func init() {
	apiBackupGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	apiBackupGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	apiBackupCmd.AddCommand(apiBackupGetCmd)
	apiBackupCmd.AddCommand(apiBackupDeleteCmd)
	apiCmd.AddCommand(apiBackupCmd)
}
