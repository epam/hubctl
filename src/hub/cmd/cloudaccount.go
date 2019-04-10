package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"hub/api"
	"hub/util"
)

var (
	getCloudCredentials          bool
	cloudCredentialsShell        bool
	cloudCredentialsNativeConfig bool
)

var cloudAccountCmd = &cobra.Command{
	Use:   "cloudaccount <get | onboard | delete> ...",
	Short: "Onboard and manage Cloud Accounts",
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

var cloudAccounOnboardCmd = &cobra.Command{
	Use:   "onboard <domain> <aws | azure | gcp> <region> <credentials...>",
	Short: "Onboard Cloud Account",
	Long: `Onboard Cloud Account to SuperHub:
<domain> must be a sub-domain of superhub.io or prefix, for example dev-01.superhub.io, dev-01

AWS:
	$ hub api onboard dev-01.superhub.io aws <access key> <secret key>

A cross account role will be created in your AWS account. The keys are not stored in SuperHub.

Azure:
	$ hub api onboard dev-01.superhub.io azure creds.json

where creds.json is a file with Service Principal credentials created by:

	$ az ad sp create-for-rbac -n <name> --sdk-auth > creds.json

This is the file usually used via AZURE_AUTH_LOCATION environment variable.
For details please consult
https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli

GCP:
	$ hub api onboard gcp dev-01.superhub.io gcp creds.json

where creds.json is a file with Service Account credentials usually used via GOOGLE_APPLICATION_CREDENTIALS environment variable.
For details please consult
https://cloud.google.com/iam/docs/creating-managing-service-accounts`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return onboardCloudAccount(args)
	},
}

var cloudAccounDeleteCmd = &cobra.Command{
	Use:   "delete <id | domain>",
	Short: "Delete Cloud Account by Id or domain",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteCloudAccount(args)
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
	api.CloudAccounts(selector, getCloudCredentials, cloudCredentialsShell, cloudCredentialsNativeConfig)

	return nil
}

func onboardCloudAccount(args []string) error {
	if len(args) < 4 || len(args) > 5 {
		return fmt.Errorf(`Onboard CloudAccount command at least four mandatory arguments:
- domain of the cloud account;
- cloud kind - one of %s;
- region;
- cloud-specific credentials.
`, strings.Join(supportedCloudAccountKinds, ","))
	}

	domain := strings.ToLower(args[0])
	if strings.Contains(domain, ".") {
		if !strings.HasSuffix(domain, SuperHubIo) {
			return fmt.Errorf("Domain `%s` must ends with `%s` or it must be a plain name", domain, SuperHubIo)
		}
	} else {
		domain += SuperHubIo
	}
	cloud := args[1]
	if !util.Contains(supportedCloudAccountKinds, cloud) {
		return fmt.Errorf("Unsupported cloud `%s`; supported clouds are: %s", cloud, strings.Join(supportedCloudAccountKinds, ","))
	}
	api.OnboardCloudAccount(domain, cloud, args[2:], waitAndTailDeployLogs)

	return nil
}

func deleteCloudAccount(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete CloudAccount command has one mandatory argument - id or domain of the cloud account")
	}

	selector := args[0]
	api.DeleteCloudAccount(selector, waitAndTailDeployLogs)

	return nil
}

func init() {
	cloudAccountGetCmd.Flags().BoolVarP(&getCloudCredentials, "cloud-credentials", "c", false,
		"Request (Temporary) Security Credentials")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudCredentialsShell, "sh", "", false,
		"Output (Temporary) Security Credentials in shell format")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudCredentialsNativeConfig, "native-config", "", false,
		"Output (Temporary) Security Credentials in cloud-specific native format: AWS CLI config or JSON")
	cloudAccounOnboardCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	cloudAccounDeleteCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	cloudAccountCmd.AddCommand(cloudAccountGetCmd)
	cloudAccountCmd.AddCommand(cloudAccounOnboardCmd)
	cloudAccountCmd.AddCommand(cloudAccounDeleteCmd)
	apiCmd.AddCommand(cloudAccountCmd)
}
