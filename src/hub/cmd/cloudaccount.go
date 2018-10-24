package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"hub/api"
)

var (
	getCloudTemporaryCredentials       bool
	cloudTemporaryCredentialsShell     bool
	cloudTemporaryCredentialsAwsConfig bool
)

var cloudAccountCmd = &cobra.Command{
	Use:   "cloudaccount <get | create | delete> ...",
	Short: "Create and manage Cloud Accounts",
}

var cloudAccountGetCmd = &cobra.Command{
	Use:   "get [id | domain]",
	Short: "Show a list of Cloud Accounts or details about the Cloud Account",
	Long: `Show a list of all user accessible Cloud Accounts or details about
the particular Cloud Account (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cloudAccount(args)
	},
}

func cloudAccount(args []string) error {
	if len(args) > 1 {
		return errors.New("CloudAccount command has one optional argument - id or domain of the cloud account")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.CloudAccounts(selector, getCloudTemporaryCredentials, cloudTemporaryCredentialsShell, cloudTemporaryCredentialsAwsConfig)

	return nil
}

func init() {
	cloudAccountGetCmd.Flags().BoolVarP(&getCloudTemporaryCredentials, "cloud-credentials", "c", false,
		"Request Temporary Security Credentials")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudTemporaryCredentialsShell, "sh", "", false,
		"Output Temporary Security Credentials in shell format")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudTemporaryCredentialsAwsConfig, "aws-config", "", false,
		"Output Temporary Security Credentials in AWS CLI config format")
	cloudAccountCmd.AddCommand(cloudAccountGetCmd)
	apiCmd.AddCommand(cloudAccountCmd)
}
