// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/api"
	"github.com/epam/hubctl/cmd/hub/util"
)

var (
	getCloudCredentials          bool
	cloudCredentialsShell        bool
	cloudCredentialsNativeConfig bool
	cfTemplateOutput             string
	cfTemplateGovCloud           bool
	awsKeypair                   string
	awsVpc                       string
	cloudZone                    string
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
	Use:   "onboard <domain> <aws | azure | gcp> <default region> [credentials...]",
	Short: "Onboard Cloud Account",
	Long: `Onboard Cloud Account to HubCTL.

Domain must be a sub-domain of epam.devops.delivery or prefix, for example dev-01.epam.devops.delivery, dev-01.

AWS:

	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-east-2 <access key> <secret key>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-east-2 <profile>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-east-2 <Role ARN>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-east-2  # credentials from OS environment, default profile, or EC2 metadata

A cross account role will be created in your AWS account. The keys are not stored in HubCTL.

AWS GovCloud:

	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-gov-east-1 <Public Cloud creds> <GovCloud access key> <GovCloud secret key>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-gov-east-1 <Public Cloud creds> <GovCloud profile>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-gov-east-1 <Public Cloud creds> <GovCloud Role ARN>
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery aws us-gov-east-1 <Public Cloud creds>  # GovCloud credentials from OS environment, default profile, or EC2 metadata

AWS GovCloud Route53 has private VPC-bound zones only, thus we need a <Public Cloud creds>
of public cloud AWS account for public DNS management.
<Public Cloud creds> is either <profile> or <access key> <secret key> pair. The supplied
credentials should be of the user with limited permissions - only on Route53.

The public cloud account could be the account associated with the GovCloud account or it
could be an independent account.

HubCTL will store public cloud keys encrypted.

Azure:

	$ hub api cloudaccount onboard dev-01.epam.devops.delivery azure eastus2 creds.json
	$ hub api cloudaccount onboard dev-01.epam.devops.delivery azure eastus2  # credentials from $AZURE_AUTH_LOCATION

where creds.json is a file with Service Principal credentials created by:

	$ az ad sp create-for-rbac -n <name> --sdk-auth > creds.json

This is the file usually used via AZURE_AUTH_LOCATION environment variable.

For details please consult
https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli
https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-authorization

GCP:

	$ hub api cloudaccount onboard gcp dev-01.epam.devops.delivery gcp us-central1 creds.json
	$ hub api cloudaccount onboard gcp dev-01.epam.devops.delivery gcp us-central1  # credentials from $GOOGLE_APPLICATION_CREDENTIALS

where creds.json is a file with Service Account credentials usually used via GOOGLE_APPLICATION_CREDENTIALS environment variable.

For details please consult
https://cloud.google.com/iam/docs/creating-managing-service-accounts
https://cloud.google.com/docs/authentication/getting-started`,
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

var cloudAccountCfTemplateCmd = &cobra.Command{
	Use:   "cf-template",
	Short: "Download AWS CloudFormation template",
	Long: `Download AWS CloudFormation template to create cross-account role:

1. Download CloudFormation template x-account-role.json. The template is specific to your AgileStack's user account.
2. Open Launch CloudFormation Stack: https://console.aws.amazon.com/cloudformation/home#/stacks/new
3. Under Choose a template section select Upload a template to Amazon S3.
4. Choose file and upload x-account-role.json.
5. Click Next.
6. Enter the Stack name.
7. Click Next.
8. Set your Options (optional) and click Next.
9. Check the I Acknowledge that AWS CloudFormation might create IAM resources box on the Review screen, and click Create.
10. When Stack Creation has completed, go to the Resources tab and click on the HubCtlRole's Physical ID.
11. Finally, copy the "Role ARN" value and paste it into:

	$ hub api cloudaccount onboard ... <Role ARN>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return downloadCfTemplate(args)
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
	api.CloudAccounts(selector, showSecrets, showLogs,
		getCloudCredentials, cloudCredentialsShell, cloudCredentialsNativeConfig, jsonFormat)

	return nil
}

func onboardCloudAccount(args []string) error {
	n := len(args)
	var cloud string
	var region string
	if n >= 3 {
		cloud = args[1]
		region = args[2]
	}

	if n < 3 || !((cloud == "aws" && n > 2 && (n < 6 || (util.Contains(api.GovCloudRegions, region) && n < 8))) ||
		(util.Contains([]string{"gcp", "azure"}, cloud) && n > 2 && n < 5)) {

		return fmt.Errorf(`Onboard Cloud Account command has at least three mandatory arguments:
- domain of the cloud account;
- cloud kind - one of %s;
- default region;
- explicit cloud-specific credentials (optional).
`, strings.Join(supportedCloudAccountKinds, ", "))
	}

	domain := strings.ToLower(args[0])
	if !strings.Contains(domain, ".") {
		domain += HubCtlApiBaseDomain
	}
	if !util.Contains(supportedCloudAccountKinds, cloud) {
		return fmt.Errorf("Unsupported cloud `%s`; supported clouds are: %s", cloud, strings.Join(supportedCloudAccountKinds, ", "))
	}
	api.OnboardCloudAccount(domain, cloud, region, args[3:], cloudZone, awsVpc, awsKeypair, waitAndTailDeployLogs)

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

func downloadCfTemplate(args []string) error {
	if len(args) != 0 {
		return errors.New("Download AWS CloudFormation template command has no arguments")
	}

	api.CloudAccountDownloadCfTemplate(cfTemplateOutput, cfTemplateGovCloud)

	return nil
}

func init() {
	cloudAccountGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "", false,
		"Show secrets")
	cloudAccountGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	cloudAccountGetCmd.Flags().BoolVarP(&getCloudCredentials, "cloud-credentials", "c", false,
		"Request (Temporary) Security Credentials")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudCredentialsShell, "sh", "", false,
		"Output (Temporary) Security Credentials in shell format")
	cloudAccountGetCmd.Flags().BoolVarP(&cloudCredentialsNativeConfig, "native-config", "", false,
		"Output (Temporary) Security Credentials in cloud-specific native format: AWS CLI config or GCP/Azure JSON")
	cloudAccountGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	cloudAccounOnboardCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	cloudAccounOnboardCmd.Flags().StringVarP(&cloudZone, "zone", "z", "",
		"Default cloud account zone (default to first zone in region)")
	cloudAccounOnboardCmd.Flags().StringVarP(&awsVpc, "aws-vpc", "", "",
		"AWS VPC Id to associate created hosted zone to (making the zone private)")
	cloudAccounOnboardCmd.Flags().StringVarP(&awsKeypair, "aws-keypair", "", "",
		"AWS EC2 SSH key-pair name (default to autogenerate new key-pair)")
	cloudAccounDeleteCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	cloudAccountCfTemplateCmd.Flags().StringVarP(&cfTemplateOutput, "output", "o", "x-account-role.json",
		"Set output filename, \"-\" for stdout")
	cloudAccountCfTemplateCmd.Flags().BoolVarP(&cfTemplateGovCloud, "govcloud", "", false,
		"GovCloud template")
	cloudAccountCmd.AddCommand(cloudAccountGetCmd)
	cloudAccountCmd.AddCommand(cloudAccounOnboardCmd)
	cloudAccountCmd.AddCommand(cloudAccounDeleteCmd)
	cloudAccountCmd.AddCommand(cloudAccountCfTemplateCmd)
	apiCmd.AddCommand(cloudAccountCmd)
}
