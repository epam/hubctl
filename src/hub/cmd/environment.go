package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"hub/api"
)

var (
	showMyTeams                  bool
	showServiceAccount           bool
	showServiceAccountLoginToken bool
)

var environmentCmd = &cobra.Command{
	Use:   "environment <get | create | delete> ...",
	Short: "Create and manage Environments",
}

var environmentGetCmd = &cobra.Command{
	Use:   "get [id | name]",
	Short: "Show a list of environments or details about the environment",
	Long: `Show a list of all user accessible environments or details about
the particular environment (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return environment(args)
	},
}

var environmentCreateCmd = &cobra.Command{
	Use:   "create <name> <cloud account id or domain>",
	Short: "Create Environment",
	Long: `Create Environment in SuperHub:
- name must be a valid DNS name
- selected Cloud Account will be attached to the Environment
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createEnvironment(args)
	},
}

var environmentDeleteCmd = &cobra.Command{
	Use:   "delete <id | name>",
	Short: "Delete Environment by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteEnvironment(args)
	},
}

func environment(args []string) error {
	if len(args) > 1 {
		return errors.New("Environment command has one optional argument - id or name of the Environment")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	if showServiceAccountLoginToken {
		showServiceAccount = true
	}
	api.Environments(selector, showSecrets, showMyTeams,
		showServiceAccount, showServiceAccountLoginToken, getCloudCredentials, jsonFormat)

	return nil
}

func createEnvironment(args []string) error {
	if len(args) != 2 {
		return errors.New("Create Environment command has two mandatory arguments - a name and id or domain of cloud account")
	}

	name := args[0]
	selector := args[1]
	api.CreateEnvironment(name, selector)

	return nil
}

var environmentPatchCmd = &cobra.Command{
	Use:   "patch <id | name> < environment-patch.json",
	Short: "Patch Environment",
	Long: `Patch Environment by sending JSON via stdin, for example:
	{
		"name": "GCP01",
		"providers": [],
		"parameters": [
			{
				"name": "component.dex.okta.appId",
				"value": "0oamwj4fg1Ih1oL0g0h7"
			},
			{
				"name": "component.dex.okta.issuer",
				"value": "https://dev-458481.oktapreview.com"
			},
			{
				"name": "component.dex.okta.clientId",
				"value": "0oamwj4fg1Ih1oL0g0h7"
			},
			{
				"kind": "secret",
				"name": "component.dex.okta.clientSecret",
				"value": {
					"kind": "token",
					"secret": "5ab6b047-ec15-4ddf-aefc-19903e6e58ed"
				}
			}
		],
		"teamsPermissions": [
			{
				"name": "ASI.Admin",
				"role": "admin"
			},
			{
				"name": "ASI.Dev",
				"role": "write"
			},
			{
				"name": "ASI.Test",
				"role": "write"
			}
		]
	}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchEnvironment(args)
	},
}

func patchEnvironment(args []string) error {
	if len(args) != 1 {
		return errors.New("Patch Environment command has one mandatory argument - id or ame of the Environment")
	}

	selector := args[0]
	if patchRaw {
		api.RawPatchEnvironment(selector, os.Stdin)
	} else {
		patchBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(patchBytes) < 3 {
			return fmt.Errorf("Unable to read patch data (read %d bytes): %v", len(patchBytes), err)
		}
		var patch api.EnvironmentPatch
		err = json.Unmarshal(patchBytes, &patch)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal patch data: %v", err)
		}
		api.PatchEnvironment(selector, patch)
	}

	return nil
}
func deleteEnvironment(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Environment command has one mandatory argument - id or name of the environment")
	}

	api.DeleteEnvironment(args[0])

	return nil
}

func init() {
	environmentGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "", false,
		"Show secrets")
	environmentGetCmd.Flags().BoolVarP(&showMyTeams, "my-teams", "m", true,
		"Show my Team(s) grants on environment")
	environmentGetCmd.Flags().BoolVarP(&showServiceAccount, "service-account", "s", false,
		"Show Service Account")
	environmentGetCmd.Flags().BoolVarP(&showServiceAccountLoginToken, "service-account-login-token", "l", false,
		"Show Service Account login token")
	environmentGetCmd.Flags().BoolVarP(&getCloudCredentials, "cloud-credentials", "c", false,
		"Request Temporary Security Credentials")
	environmentGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	environmentPatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	environmentCmd.AddCommand(environmentGetCmd)
	environmentCmd.AddCommand(environmentCreateCmd)
	environmentCmd.AddCommand(environmentPatchCmd)
	environmentCmd.AddCommand(environmentDeleteCmd)
	apiCmd.AddCommand(environmentCmd)
}
