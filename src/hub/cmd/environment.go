package cmd

import (
	"errors"

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
	environmentCmd.AddCommand(environmentGetCmd)
	environmentCmd.AddCommand(environmentCreateCmd)
	environmentCmd.AddCommand(environmentDeleteCmd)
	apiCmd.AddCommand(environmentCmd)
}
