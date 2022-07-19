// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
)

var applicationCmd = &cobra.Command{
	Use:   "application <get | install | patch | delete> ...",
	Short: "Create and manage Applications",
}

var applicationGetCmd = &cobra.Command{
	Use:   "get [id | name]",
	Short: "Show a list of applications or details about the application",
	Long: `Show a list of all user accessible applications or details about
the particular application (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return application(args)
	},
}

var applicationInstallCmd = &cobra.Command{
	Use:   "install < application.json",
	Short: "Install Application",
	Long: fmt.Sprintf(`Install Application by sending JSON via stdin, for example:
%[1]s
	{
		"name": "a-node-01",
		"description": "NodeJS microservice with Express",
		"application": "nodejs-backend",
		"requires": ["jenkins", "github", "harbor"],
		"platform": "2",
		"parameters": [
			{
				"name": "application.name",
				"value": "a-node-02"
			},
			...
		}]
	}
%[1]s`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return installApplication(args)
	},
}

var applicationPatchCmd = &cobra.Command{
	Use:   "patch <id | name> < application-patch.json",
	Short: "Patch Application",
	Long: fmt.Sprintf(`Patch Application by sending JSON via stdin, for example:
%[1]s
	{
		"parameters": [
			{
				"name": "application.replicas",
				"value": 3
			}
		]
	}
%[1]s`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchApplication(args)
	},
}

var applicationDeleteCmd = &cobra.Command{
	Use:   "delete <id | name>",
	Short: "Delete Application",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteApplication(args)
	},
}

func application(args []string) error {
	if len(args) > 1 {
		return errors.New("Application command has one optional argument - id or name of the application")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Applications(selector, showSecrets, showLogs, jsonFormat)

	return nil
}

func installApplication(args []string) error {
	if len(args) > 0 {
		return errors.New("Install Application command has no arguments")
	}

	if createRaw {
		api.RawInstallApplication(os.Stdin, waitAndTailDeployLogs)
	} else {
		createBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(createBytes) < 3 {
			return fmt.Errorf("Unable to read data (read %d bytes): %v", len(createBytes), err)
		}
		var req api.ApplicationRequest
		err = json.Unmarshal(createBytes, &req)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal data: %v", err)
		}
		api.InstallApplication(req, waitAndTailDeployLogs)
	}

	return nil
}

func patchApplication(args []string) error {
	if len(args) != 1 {
		return errors.New("Patch Application command has one mandatory argument - id or name of the Application")
	}

	selector := args[0]
	if patchRaw {
		api.RawPatchApplication(selector, os.Stdin, waitAndTailDeployLogs)
	} else {
		patchBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(patchBytes) < 3 {
			return fmt.Errorf("Unable to read patch data (read %d bytes): %v", len(patchBytes), err)
		}
		var patch api.ApplicationPatch
		err = json.Unmarshal(patchBytes, &patch)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal patch data: %v", err)
		}
		api.PatchApplication(selector, patch, waitAndTailDeployLogs)
	}

	return nil
}

func deleteApplication(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Application command has one mandatory argument - id or name of the Application")
	}

	api.DeleteApplication(args[0], waitAndTailDeployLogs)

	return nil
}

func init() {
	applicationGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "s", false,
		"Show secrets")
	applicationGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	applicationGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	applicationInstallCmd.Flags().BoolVarP(&createRaw, "raw", "r", false,
		"Send data as is, do not trim non-POST-able API object fields")
	applicationPatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	applicationInstallCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for install to complete and tail logs")
	applicationPatchCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for update to complete and tail logs")
	applicationDeleteCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for delete to complete and tail logs")
	applicationCmd.AddCommand(applicationGetCmd)
	applicationCmd.AddCommand(applicationInstallCmd)
	applicationCmd.AddCommand(applicationPatchCmd)
	applicationCmd.AddCommand(applicationDeleteCmd)
	apiCmd.AddCommand(applicationCmd)
}
