package api

import (
	"fmt"
	"log"
	"net/url"
	"strconv"

	"hub/config"
)

const cloudAccountsResource = "hub/api/v1/cloud-accounts"

var cloudAccountsCache = make(map[string]*CloudAccount)

func CloudAccounts(selector string,
	getCloudTemporaryCredentials, shFormat, awsConfigFormat bool) {

	cloudAccounts, err := cloudAccountsBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Cloud Account(s): %v", err)
	}
	if getCloudTemporaryCredentials && (shFormat || awsConfigFormat) {
		if len(cloudAccounts) == 0 {
			fmt.Print("# No Cloud Accounts\n")
		} else {
			errors := make([]error, 0)
			for i, cloudAccount := range cloudAccounts {
				keys, err := cloudAccountTemporaryCredentials(cloudAccount.Id)
				if err != nil {
					errors = append(errors, err)
				} else {
					sh := ""
					if shFormat {
						sh = formatCloudAccountTemporaryCredentialsSh(keys)
					}
					awsConfig := ""
					if awsConfigFormat {
						awsConfig = formatCloudAccountTemporaryCredentialsAwsConfig(&cloudAccount, keys)
					}
					if i > 0 {
						fmt.Print("\n")
					}
					fmt.Printf("# %s%s%s", cloudAccount.BaseDomain, sh, awsConfig)
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
			fmt.Printf("\t\tKind: %s\n", formatCloudAccountKind(cloudAccount.Type))
			fmt.Printf("\t\tStatus: %s\n", cloudAccount.Status)
			if getCloudTemporaryCredentials {
				keys, err := cloudAccountTemporaryCredentials(cloudAccount.Id)
				if err != nil {
					errors = append(errors, err)
				} else {
					fmt.Printf("\t\tTemporary Security Credentials: %s\n", formatCloudAccountTemporaryCredentials(keys))
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
				formatted, err := formatParameter(resource, param)
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

var cloudAccountKinds = map[string]string{
	"aws":    "access and secret keys",
	"awscar": "automatic cross-account IAM role",
	"awsarn": "manually entered cross-account IAM role ARN",
}

func formatCloudAccountKind(kind string) string {
	return fmt.Sprintf("%s (%s)", kind, cloudAccountKinds[kind])
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

func cloudAccountTemporaryCredentials(id string) (*AwsTemporarySecurityCredentials, error) {
	if config.Debug {
		log.Printf("Getting Cloud Account `%s` temporary security credentials", id)
	}
	path := fmt.Sprintf("%s/%s/session-keys", cloudAccountsResource, url.PathEscape(id))
	var jsResp AwsTemporarySecurityCredentials
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Cloud Account `%s` Temporary Security Credentials: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Cloud Account `%s` Temporary Security Credentials, expected 200 HTTP",
			code, id)
	}
	return &jsResp, nil
}

func formatCloudAccountTemporaryCredentials(keys *AwsTemporarySecurityCredentials) string {
	return fmt.Sprintf("%s ttl = %d\n\t\t\tAccess = %s\n\t\t\tSecret = %s\n\t\t\tSession = %s",
		keys.Cloud, keys.Ttl,
		keys.AccessKey, keys.SecretKey, keys.SessionToken)
}

func formatCloudAccountTemporaryCredentialsSh(keys *AwsTemporarySecurityCredentials) string {
	return fmt.Sprintf("\n# eval this in your shell\nexport AWS_ACCESS_KEY_ID=%s\nexport AWS_SECRET_ACCESS_KEY=%s\nexport AWS_SESSION_TOKEN=%s\n",
		keys.AccessKey, keys.SecretKey, keys.SessionToken)
}

func formatCloudAccountTemporaryCredentialsAwsConfig(account *CloudAccount, keys *AwsTemporarySecurityCredentials) string {
	return fmt.Sprintf("\n[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\naws_session_token = %s\n",
		account.BaseDomain, keys.AccessKey, keys.SecretKey, keys.SessionToken)
}
