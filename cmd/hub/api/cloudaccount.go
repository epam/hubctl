// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/epam/hubctl/cmd/hub/aws"
	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

const cloudAccountsResource = "hub/api/v1/cloud-accounts"

var (
	GovCloudRegions = []string{"us-gov-east-1", "us-gov-west-1"}
	//lint:ignore U1000 still needed?
	cloudAccountsCache = make(map[string]*CloudAccount)
)

func CloudAccounts(selector string, showSecrets, showLogs,
	getCloudCredentials, shFormat, nativeConfigFormat, jsonFormat bool) {

	cloudAccounts, err := cloudAccountsBy(selector, showSecrets)
	if err != nil {
		log.Fatalf("Unable to query for Cloud Account(s): %v", err)
	}
	if getCloudCredentials && (shFormat || nativeConfigFormat) {
		if len(cloudAccounts) == 0 {
			fmt.Print("# No Cloud Accounts\n")
		} else {
			errors := make([]error, 0)
			for i, cloudAccount := range cloudAccounts {
				keys, err := cloudAccountCredentials(cloudAccount.Id, cloudAccount.Kind)
				if err != nil {
					errors = append(errors, err)
				} else {
					sh := ""
					if shFormat {
						sh, err = formatCloudAccountCredentialsSh(keys)
						if err != nil {
							errors = append(errors, err)
						}
					}
					nativeConfig := ""
					if nativeConfigFormat {
						nativeConfig, err = formatCloudAccountCredentialsNativeConfig(&cloudAccount, keys)
						if err != nil {
							errors = append(errors, err)
						}
					}
					if i > 0 {
						fmt.Print("\n")
					}
					header := ""
					if len(cloudAccounts) > 1 {
						header = fmt.Sprintf("# %s\n", cloudAccount.BaseDomain)
					}
					if sh != "" || nativeConfig != "" {
						fmt.Printf("%s%s%s", header, sh, nativeConfig)
					}
				}
			}
			if len(errors) > 0 {
				fmt.Print("# Errors encountered:\n")
				for _, err := range errors {
					fmt.Printf("#\t%s\n", strings.ReplaceAll(err.Error(), "\n", "\n#\t"))
				}
			}
		}
	} else if len(cloudAccounts) == 0 {
		if jsonFormat {
			log.Print("No Cloud Accounts")
		} else {
			fmt.Print("No Cloud Accounts\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(cloudAccounts) == 1 {
				toMarshal = &cloudAccounts[0]
			} else {
				toMarshal = cloudAccounts
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Cloud Accounts:\n")
			errors := make([]error, 0)
			for _, cloudAccount := range cloudAccounts {
				errors = formatCloudAccountEntity(&cloudAccount, getCloudCredentials, showSecrets, showLogs, errors)
			}
			if len(errors) > 0 {
				fmt.Print("Errors encountered:\n")
				for _, err := range errors {
					fmt.Printf("\t%v\n", err)
				}
			}
		}
	}
}

func formatCloudAccountEntity(cloudAccount *CloudAccount, getCloudCredentials, showSecrets, showLogs bool, errors []error) []error {
	fmt.Printf("\n\t%s\n", formatCloudAccountTitle(cloudAccount))
	fmt.Printf("\t\tKind: %s\n", formatCloudAccountKind(cloudAccount.Kind))
	fmt.Printf("\t\tStatus: %s\n", cloudAccount.Status)
	if getCloudCredentials {
		keys, err := cloudAccountCredentials(cloudAccount.Id, cloudAccount.Kind)
		if err != nil {
			errors = append(errors, err)
		} else {
			formatted, err := formatCloudAccountCredentials(keys)
			if err != nil {
				errors = append(errors, err)
			} else {
				fmt.Printf("\t\tSecurity Credentials: %s\n", formatted)
			}
		}
	}
	if len(cloudAccount.TeamsPermissions) > 0 {
		formatted := formatTeams(cloudAccount.TeamsPermissions)
		fmt.Printf("\t\tTeams: %s\n", formatted)
	}
	if len(cloudAccount.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
	}
	resource := fmt.Sprintf("%s/%s", cloudAccountsResource, cloudAccount.Id)
	for _, param := range sortParameters(cloudAccount.Parameters) {
		formatted, err := formatParameter(resource, param, showSecrets)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(cloudAccount.InflightOperations) > 0 {
		fmt.Print("\t\tInflight Operations:\n")
		for _, op := range cloudAccount.InflightOperations {
			fmt.Print(formatInflightOperation(op, showLogs))
		}
	}
	return errors
}

func formatCloudAccount(cloudAccount *CloudAccount) {
	errors := formatCloudAccountEntity(cloudAccount, false, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

//lint:ignore U1000 still needed?
func cachedCloudAccountBy(selector string) (*CloudAccount, error) {
	cloudAccount, cached := cloudAccountsCache[selector]
	if !cached {
		var err error
		cloudAccount, err = cloudAccountBy(selector)
		if err != nil {
			return nil, err
		}
		cloudAccountsCache[selector] = cloudAccount
	}
	return cloudAccount, nil
}

func cloudAccountBy(selector string) (*CloudAccount, error) {
	if !util.IsUint(selector) {
		return cloudAccountByDomain(selector)
	}
	return cloudAccountById(selector, false)
}

func cloudAccountsBy(selector string, unmask bool) ([]CloudAccount, error) {
	if !util.IsUint(selector) {
		return cloudAccountsByDomain(selector)
	}
	cloudAccount, err := cloudAccountById(selector, unmask)
	if err != nil {
		return nil, err
	}
	if cloudAccount != nil {
		return []CloudAccount{*cloudAccount}, nil
	}
	return nil, nil
}

func cloudAccountById(id string, unmask bool) (*CloudAccount, error) {
	maybeUnmask := ""
	if unmask {
		maybeUnmask = "?unmask=true"
	}
	path := fmt.Sprintf("%s/%s%s", cloudAccountsResource, url.PathEscape(id), maybeUnmask)
	var jsResp CloudAccount
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying HubCTL Cloud Accounts: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying HubCTL Cloud Accounts, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func cloudAccountByDomain(domain string) (*CloudAccount, error) {
	cloudAccounts, err := cloudAccountsByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Cloud Account `%s`: %v", domain, err)
	}
	if len(cloudAccounts) == 0 {
		return nil, fmt.Errorf("No Cloud Account `%s` found", domain)
	}
	if len(cloudAccounts) > 1 {
		return nil, fmt.Errorf("More than one Cloud Account returned by domain `%s`", domain)
	}
	cloudAccount := cloudAccounts[0]
	return &cloudAccount, nil
}

func cloudAccountsByDomain(domain string) ([]CloudAccount, error) {
	path := cloudAccountsResource
	if domain != "" {
		path += "?domain=" + url.QueryEscape(domain)
	}
	var jsResp []CloudAccount
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying HubCTL Cloud Accounts: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying HubCTL Cloud Accounts, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatCloudAccountTitle(account *CloudAccount) string {
	return fmt.Sprintf("%s / %s [%s]", account.Name, account.BaseDomain, account.Id)
}

var cloudAccountKindDescription = map[string]string{
	"aws":    "AWS access and secret keys",
	"awscar": "AWS automatic cross-account IAM role",
	"awsarn": "AWS manually entered cross-account IAM role ARN",
	"azure":  "Azure",
	"gcp":    "Google Cloud Platform",
}

func formatCloudAccountKind(kind string) string {
	description := cloudAccountKindDescription[kind]
	if description != "" {
		description = fmt.Sprintf(" (%s)", description)
	}
	return fmt.Sprintf("%s%s", kind, description)
}

func cloudAccountRegion(account *CloudAccount) string {
	for _, p := range account.Parameters {
		if p.Name == "cloud.region" {
			if maybeStr, ok := p.Value.(string); ok {
				return maybeStr
			}
		}
	}
	return ""
}

func cloudAccountCredentials(id, kind string) (interface{}, error) {
	switch kind {
	case "aws", "awscar", "awsarn":
		return awsCloudAccountCredentials(id)
	case "azure", "gcp":
		return rawCloudAccountCredentials(id)
	}
	return nil, fmt.Errorf("Unsupported cloud account kind `%s`", kind)
}

func rawCloudAccountCredentials(id string) ([]byte, error) {
	if config.Debug {
		log.Printf("Getting Cloud Account `%s` credentials", id)
	}
	path := fmt.Sprintf("%s/%s/session-keys", cloudAccountsResource, url.PathEscape(id))
	code, body, err := get2(hubApi(), path)
	if err != nil {
		return nil, fmt.Errorf("Error querying HubCTL Cloud Account `%s` Credentials: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying HubCTL Cloud Account `%s` Credentials, expected 200 HTTP",
			code, id)
	}
	return body, nil
}

func awsCloudAccountCredentials(id string) (*AwsSecurityCredentials, error) {
	if config.Debug {
		log.Printf("Getting Cloud Account `%s` credentials", id)
	}
	path := fmt.Sprintf("%s/%s/session-keys", cloudAccountsResource, url.PathEscape(id))
	var jsResp AwsSecurityCredentials
	code, err := get(hubApi(), path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying HubCTL Cloud Account `%s` Credentials: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying HubCTL Cloud Account `%s` Credentials, expected 200 HTTP",
			code, id)
	}
	return &jsResp, nil
}

func formatCloudAccountCredentials(keys interface{}) (string, error) {
	if aws, ok := keys.(*AwsSecurityCredentials); ok {
		return formatAwsCloudAccountCredentials(aws)
	}
	if raw, ok := keys.([]byte); ok {
		return formatRawCloudAccountCredentialsSh(raw)
	}
	return "", fmt.Errorf("Unable to format credentials: %+v", keys)
}

func formatCloudAccountCredentialsSh(keys interface{}) (string, error) {
	if aws, ok := keys.(*AwsSecurityCredentials); ok {
		return formatAwsCloudAccountCredentialsSh(aws)
	}
	if raw, ok := keys.([]byte); ok {
		return formatRawCloudAccountCredentialsSh(raw)
	}
	return "", fmt.Errorf("Unable to format credentials: %+v", keys)
}

func formatCloudAccountCredentialsNativeConfig(account *CloudAccount, keys interface{}) (string, error) {
	if aws, ok := keys.(*AwsSecurityCredentials); ok {
		return formatAwsCloudAccountCredentialsCliConfig(account, aws)
	}
	if raw, ok := keys.([]byte); ok {
		return formatRawCloudAccountCredentialsSh(raw)
	}
	return "", fmt.Errorf("Unable to format credentials: %+v", keys)
}

func formatAwsCloudAccountCredentials(keys *AwsSecurityCredentials) (string, error) {
	maybeSts := ""
	if keys.Sts != "" {
		maybeSts = "; sts = " + keys.Sts
	}
	maybeRegion := ""
	if keys.Region != "" {
		maybeRegion = "; region = " + keys.Region
	}
	return fmt.Sprintf("%s ttl = %d%s%s\n\t\t\tAccess = %s\n\t\t\tSecret = %s\n\t\t\tSession = %s",
		keys.Cloud, keys.Ttl, maybeSts, maybeRegion,
		keys.AccessKey, keys.SecretKey, keys.SessionToken), nil
}

func formatAwsCloudAccountCredentialsSh(keys *AwsSecurityCredentials) (string, error) {
	maybeSts := ""
	if keys.Sts != "" {
		maybeSts = "\n# sts = " + keys.Sts
	}
	maybeRegion := ""
	if keys.Region != "" {
		maybeRegion = "\nexport AWS_DEFAULT_REGION=" + keys.Region
	}
	return fmt.Sprintf(`# eval this in your shell
# ttl = %d%s%s
export AWS_ACCESS_KEY_ID=%s
export AWS_SECRET_ACCESS_KEY=%s
export AWS_SESSION_TOKEN=%s
`,
		keys.Ttl, maybeSts, maybeRegion, keys.AccessKey, keys.SecretKey, keys.SessionToken), nil
}

func formatAwsCloudAccountCredentialsCliConfig(account *CloudAccount, keys *AwsSecurityCredentials) (string, error) {
	return fmt.Sprintf("[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n",
		account.BaseDomain, keys.AccessKey, keys.SecretKey, keys.SessionToken), nil
}

func formatRawCloudAccountCredentialsSh(raw []byte) (string, error) {
	var kv map[string]interface{}
	err := json.Unmarshal(raw, &kv)
	if err != nil {
		return "", fmt.Errorf("Unable un unmarshal: %v", err)
	}
	lines := make([]string, 0, len(kv))
	for k, v := range kv {
		line := fmt.Sprintf("%s=%v", k, v)
		lines = append(lines, line)
	}
	ident := "\n\t\t\t"
	return ident + strings.Join(lines, ident), nil
}

func OnboardCloudAccount(domain, kind, region string, args []string, zone, awsVpc, awsKeypair string, waitAndTailDeployLogs bool) {
	err := onboardCloudAccount(domain, kind, region, args, zone, awsVpc, awsKeypair, waitAndTailDeployLogs)
	if err != nil {
		log.Fatalf("Unable to onboard Cloud Account: %v", err)
	}
}

func onboardCloudAccount(domain, kind, region string, args []string, zone, awsVpc, awsKeypair string, waitAndTailDeployLogs bool) error {
	kind2, credentials, err := cloudSpecificCredentials(kind, region, args)
	if err != nil {
		return err
	}

	if config.Debug {
		log.Printf("Onboarding %s cloud account %s in %s with %v", kind, domain, region, args)
	}

	provider := kind
	domainParts := strings.SplitN(domain, ".", 2)
	if len(domainParts) != 2 {
		return fmt.Errorf("Domain `%s` is invalid", domain)
	}
	if zone == "" {
		zone = cloudFirstZoneInRegion(provider, region)
	} else if !strings.HasPrefix(zone, region) {
		return fmt.Errorf("Zone `%s` is not within `%s` region", zone, region)
	}
	parameters := []Parameter{
		{Name: "dns.baseDomain", Value: domainParts[1]},
		{Name: "cloud.provider", Value: provider},
		{Name: "cloud.region", Value: region},
		{Name: "cloud.availabilityZone", Value: zone},
	}
	if provider == "aws" {
		if awsVpc != "" {
			parameters = append(parameters, Parameter{Name: "cloud.vpc", Value: awsVpc})
		}
		if awsKeypair != "" {
			parameters = append(parameters, Parameter{Name: "cloud.sshKey", Value: awsKeypair})
		}
	}
	req := &CloudAccountRequest{
		Name:        domainParts[0],
		Kind:        kind2,
		Credentials: credentials,
		Parameters:  parameters,
	}
	if provider == "azure" {
		req.Parameters = append(req.Parameters, Parameter{Name: "cloud.azureResourceGroupName", Value: "hubctl-" + region})
	}

	account, err := createCloudAccount(req)
	if err != nil {
		return err
	}
	formatCloudAccount(account)
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"cloudAccount/" + domain}, true))
	}
	return nil
}

func cloudSpecificCredentials(provider, region string, args []string) (string, map[string]string, error) {
	switch provider {
	case "aws":
		var kind string
		credentials := make(map[string]string)
		if util.Contains(GovCloudRegions, region) && len(args) >= 1 {
			if maybeAwsAccessKey(args[0]) && len(args) >= 2 {
				credentials["dnsAccessKey"] = args[0]
				credentials["dnsSecretKey"] = args[1]
				args = args[2:]
			} else {
				profile := args[0]
				creds, err := awsCredentials(profile)
				if err != nil {
					return "", nil, err
				}
				if creds.SessionToken != "" {
					return "", nil, fmt.Errorf("AWS credentials retrieved has session token set (profile `%s`)", profile)
				}
				credentials["dnsAccessKey"] = creds.AccessKeyID
				credentials["dnsSecretKey"] = creds.SecretAccessKey
				args = args[1:]
			}
		}
		if len(args) == 2 {
			kind = "awscar"
			credentials["accessKey"] = args[0]
			credentials["secretKey"] = args[1]
		} else if len(args) == 1 && strings.HasPrefix(args[0], "arn:aws") {
			kind = "awsarn"
			credentials["roleArn"] = args[0]
		} else {
			profile := ""
			if len(args) == 1 {
				profile = args[0]
			}
			creds, err := awsCredentials(profile)
			if err != nil {
				return "", nil, err
			}
			kind = "awscar"
			credentials["accessKey"] = creds.AccessKeyID
			credentials["secretKey"] = creds.SecretAccessKey
			credentials["sessionToken"] = creds.SessionToken
		}
		return kind, credentials, nil

	case "azure", "gcp":
		credentialsFile := ""
		if len(args) == 1 {
			credentialsFile = args[0]
		}
		if credentialsFile == "" {
			if provider == "gcp" {
				credentialsFile = config.GcpCredentialsFile
				if credentialsFile == "" {
					credentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
				}
			} else if provider == "azure" {
				credentialsFile = config.AzureCredentialsFile
				if credentialsFile == "" {
					credentialsFile = os.Getenv("AZURE_AUTH_LOCATION")
				}
			}
		}
		if credentialsFile == "" {
			return "", nil, errors.New("No credentials file specified")
		}
		file, err := os.Open(credentialsFile)
		if err != nil {
			return "", nil, fmt.Errorf("Unable to open credentials file: %v", err)
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return "", nil, fmt.Errorf("Unable to read credentials file `%s`: %v", credentialsFile, err)
		}
		var creds map[string]string
		err = json.Unmarshal(data, &creds)
		if err != nil {
			return "", nil, fmt.Errorf("Unable to unmarshall credentials file `%s`: %v", credentialsFile, err)
		}
		return provider, creds, nil
	}
	return "", nil, errors.New("Unknown cloud account provider")
}

// can backfire?
// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AccessKey.html
func maybeAwsAccessKey(key string) bool {
	return len(key) == 20 && strings.HasPrefix(key, "AK")
}

func awsCredentials(profile string) (*awscredentials.Value, error) {
	savePref := config.AwsPreferProfileCredentials // TODO
	if profile != "" {
		config.AwsPreferProfileCredentials = true
	}
	factory := aws.ProfileCredentials(profile, "cloud account onboarding")
	creds, err := factory.Get()
	config.AwsPreferProfileCredentials = savePref
	if err != nil {
		maybeProfile := ""
		if profile != "" {
			maybeProfile = fmt.Sprintf(" (profile `%s`)", profile)
		}
		return nil, fmt.Errorf("Unable to retrieve AWS credentials%s: %v", maybeProfile, err)
	}
	return &creds, nil
}

func cloudFirstZoneInRegion(provider, region string) string {
	switch provider {
	case "aws":
		return region + "a"
	case "azure":
		return "1"
	case "gcp":
		// https://cloud.google.com/compute/docs/regions-zones/
		if util.Contains([]string{"europe-west1", "us-east1"}, region) {
			return region + "-b"
		}
		return region + "-a"
	}
	return ""
}

func createCloudAccount(cloudAccount *CloudAccountRequest) (*CloudAccount, error) {
	var jsResp CloudAccount
	code, err := post(hubApi(), cloudAccountsResource, cloudAccount, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 && code != 202 {
		return nil, fmt.Errorf("Got %d HTTP creating HubCTL Cloud Account, expected [200, 201, 202] HTTP", code)
	}
	return &jsResp, nil
}

func DeleteCloudAccount(selector string, waitAndTailDeployLogs bool) {
	if config.Debug {
		log.Printf("Deleting %s cloud account", selector)
	}
	code, err := deleteCloudAccount(selector)
	if err != nil {
		log.Fatalf("Unable to delete HubCTL Cloud Account: %v", err)
	}
	if waitAndTailDeployLogs && code == 202 {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"cloudAccount/" + selector}, true))
	}
}

func deleteCloudAccount(selector string) (int, error) {
	account, err := cloudAccountBy(selector)
	id := ""
	if err != nil {
		str := err.Error()
		if util.IsUint(selector) &&
			(strings.Contains(str, "json: cannot unmarshal") || strings.Contains(str, "cannot parse") || config.Force) {
			util.Warn("%v", err)
			id = selector
		} else {
			return 0, err
		}
	} else if account == nil {
		return 404, error404
	} else {
		id = account.Id
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", cloudAccountsResource, url.PathEscape(id), force)
	code, err := delete(hubApi(), path)
	if err != nil {
		return code, err
	}
	if code != 202 && code != 204 {
		return code, fmt.Errorf("Got %d HTTP deleting HubCTL Cloud Account, expected [202, 204] HTTP", code)
	}
	return code, nil
}

func CloudAccountDownloadCfTemplate(filename string, govcloud bool) {
	err := cloudAccountDownloadCfTemplate(filename, govcloud)
	if err != nil {
		log.Fatalf("Unable to download HubCTL Cloud Account AWS CloudFormation template: %v", err)
	}
}

func cloudAccountDownloadCfTemplate(filename string, govcloud bool) error {
	path := fmt.Sprintf("%s/x-account-role-template-download", cloudAccountsResource)
	if govcloud {
		path = fmt.Sprintf("%s?govcloud=true", path)
	}
	code, body, err := get2(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("Got %d HTTP fetching HubCTL Cloud Account AWS CloudFormation template, expected 200 HTTP", code)
	}
	if len(body) == 0 {
		return fmt.Errorf("Got empty HubCTL Cloud Account AWS CloudFormation template")
	}

	var file io.WriteCloser
	if filename == "-" {
		file = os.Stdout
	} else {
		info, _ := os.Stat(filename)
		if info != nil {
			if info.IsDir() {
				filename = fmt.Sprintf("%s/x-account-role.json", filename)
			} else {
				if !config.Force {
					log.Fatalf("File `%s` exists, use --force / -f to overwrite", filename)
				}
			}
		}
		var err error
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("Unable to create %s: %v", filename, err)
		}
		defer file.Close()
	}
	written, err := file.Write(body)
	if written != len(body) {
		return fmt.Errorf("Unable to write %s: %v", filename, err)
	}
	if config.Verbose && filename != "-" {
		log.Printf("Wrote %s", filename)
	}

	return nil
}
