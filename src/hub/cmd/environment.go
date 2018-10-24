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
	api.Environments(selector, showMyTeams, showServiceAccount, showServiceAccountLoginToken, getCloudTemporaryCredentials)

	return nil
}

func init() {
	environmentGetCmd.Flags().BoolVarP(&showMyTeams, "my-teams", "m", true,
		"Show my Team(s) grants on environment")
	environmentGetCmd.Flags().BoolVarP(&showServiceAccount, "service-account", "s", false,
		"Show Service Account")
	environmentGetCmd.Flags().BoolVarP(&showServiceAccountLoginToken, "service-account-login-token", "l", false,
		"Show Service Account login token")
	environmentGetCmd.Flags().BoolVarP(&getCloudTemporaryCredentials, "cloud-credentials", "c", false,
		"Request Temporary Security Credentials")
	environmentCmd.AddCommand(environmentGetCmd)
	apiCmd.AddCommand(environmentCmd)
}
