package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"hub/config"
	"hub/util"
)

const cloudAccountsResource = "hub/api/v1/cloud-accounts"

var cloudAccountsCache = make(map[string]*CloudAccount)

func CloudAccounts(selector string,
	getCloudCredentials, shFormat, nativeConfigFormat bool) {

	cloudAccounts, err := cloudAccountsBy(selector)
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
					fmt.Printf("# \t%v\n", err)
				}
			}
		}
	} else if len(cloudAccounts) == 0 {
		fmt.Print("No Cloud Accounts\n")
	} else {
		fmt.Print("Cloud Accounts:\n")
		errors := make([]error, 0)
		for _, cloudAccount := range cloudAccounts {
			fmt.Printf("\n\t%s\n", formatCloudAccount(&cloudAccount))
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
				formatted, err := formatParameter(resource, param, false)
				fmt.Printf("\t\t%s\n", formatted)
				if err != nil {
					errors = append(errors, err)
				}
			}
		}
		if len(errors) > 0 {
			fmt.Print("Errors encountered:\n")
			for _, err := range errors {
				fmt.Printf("\t%v\n", err)
			}
		}
	}
}

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
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return cloudAccountByDomain(selector)
	}
	return cloudAccountById(selector)
}

func cloudAccountsBy(selector string) ([]CloudAccount, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return cloudAccountsByDomain(selector)
	}
	cloudAccount, err := cloudAccountById(selector)
	if err != nil {
		return nil, err
	}
	if cloudAccount != nil {
		return []CloudAccount{*cloudAccount}, nil
	}
	return nil, nil
}

func cloudAccountById(id string) (*CloudAccount, error) {
	path := fmt.Sprintf("%s/%s", cloudAccountsResource, url.PathEscape(id))
	var jsResp CloudAccount
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Cloud Accounts: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Cloud Accounts, expected 200 HTTP", code)
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
	instance := cloudAccounts[0]
	return &instance, nil
}

func cloudAccountsByDomain(domain string) ([]CloudAccount, error) {
	path := cloudAccountsResource
	if domain != "" {
		path += "?domain=" + url.QueryEscape(domain)
	}
	var jsResp []CloudAccount
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Cloud Accounts: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Cloud Accounts, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatCloudAccount(account *CloudAccount) string {
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
	code, err, body := get2(hubApi, path)
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Cloud Account `%s` Credentials: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Cloud Account `%s` Credentials, expected 200 HTTP",
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
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Cloud Account `%s` Credentials: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Cloud Account `%s` Credentials, expected 200 HTTP",
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
	return fmt.Sprintf("%s ttl = %d\n\t\t\tAccess = %s\n\t\t\tSecret = %s\n\t\t\tSession = %s",
		keys.Cloud, keys.Ttl,
		keys.AccessKey, keys.SecretKey, keys.SessionToken), nil
}

func formatAwsCloudAccountCredentialsSh(keys *AwsSecurityCredentials) (string, error) {
	return fmt.Sprintf("# eval this in your shell\nexport AWS_ACCESS_KEY_ID=%s\nexport AWS_SECRET_ACCESS_KEY=%s\nexport AWS_SESSION_TOKEN=%s\n",
		keys.AccessKey, keys.SecretKey, keys.SessionToken), nil
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

func formatRawCloudAccountCredentialsNativeConfig(raw []byte) (string, error) {
	return string(raw), nil
}

func OnboardCloudAccount(domain, kind string, args []string, waitAndTailDeployLogs bool) {
	credentials, err := cloudSpecificCredentials(kind, args)
	if err != nil {
		log.Fatalf("Unable to onboard Cloud Account: %v", err)
	}

	if config.Debug {
		log.Printf("Onboarding %s cloud account %s with %v", kind, domain, args)
	}

	provider := kind
	if kind == "aws" {
		kind = "awscar" // cross-account role
	}
	domainParts := strings.SplitN(domain, ".", 2)
	if len(domainParts) != 2 {
		log.Fatalf("Domain `%s` invalid", domain)
	}
	region := args[0]
	req := &CloudAccountRequest{
		Name:        domainParts[0],
		Kind:        kind,
		Credentials: credentials,
		Parameters: []Parameter{
			{Name: "dns.baseDomain", Value: domainParts[1]},
			{Name: "cloud.provider", Value: provider},
			{Name: "cloud.region", Value: region},
			{Name: "cloud.availabilityZone", Value: cloudFirstZoneInRegion(provider, region)},
			// {Name: "cloud.sshKey", Value: "agilestacks"},
		},
	}
	if provider == "azure" {
		req.Parameters = append(req.Parameters, Parameter{Name: "cloud.azureResourceGroupName", Value: "superhub-" + region})
	}

	account, err := createCloudAccount(req)
	if err != nil {
		log.Fatalf("Unable to onboard Cloud Account: %v", err)
	}
	CloudAccounts(account.Id, false, false, false)
	if waitAndTailDeployLogs {
		// TODO wait for cloudAccount status update and exit on success of failure
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		Logs([]string{"cloudAccount/" + domain})
	}
}

func cloudSpecificCredentials(provider string, args []string) (map[string]string, error) {
	switch provider {
	case "aws":
		return map[string]string{"accessKey": args[1], "secretKey": args[2]}, nil

	case "azure", "gcp":
		credentialsFile := args[1]
		file, err := os.Open(credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("Unable to open credentials file: %v", err)
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("Unable to read credentials file `%s`: %v", credentialsFile, err)
		}
		var creds map[string]string
		err = json.Unmarshal(data, &creds)
		if err != nil {
			return nil, fmt.Errorf("Unable to unmarshall credentials file `%s`: %v", credentialsFile, err)
		}
		return creds, nil
	}
	return nil, nil
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
	code, err := post(hubApi, cloudAccountsResource, cloudAccount, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 && code != 202 {
		return nil, fmt.Errorf("Got %d HTTP creating Hub Service Cloud Account, expected [200, 201, 202] HTTP", code)
	}
	return &jsResp, nil
}

func DeleteCloudAccount(selector string, waitAndTailDeployLogs bool) {
	if config.Debug {
		log.Printf("Deleting %s cloud account", selector)
	}
	err := deleteCloudAccount(selector)
	if err != nil {
		log.Fatalf("Unable to delete Hub Service Cloud Account: %v", err)
	}
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		Logs([]string{"cloudAccount/" + selector})
	}
}

func deleteCloudAccount(selector string) error {
	account, err := cloudAccountBy(selector)
	if err != nil {
		return err
	}
	if account == nil {
		return error404
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", cloudAccountsResource, url.PathEscape(account.Id), force)
	code, err := delete(hubApi, path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting Hub Service Cloud Account, expected [202, 204] HTTP", code)
	}
	return nil
}
