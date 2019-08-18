package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const environmentsResource = "hub/api/v1/environments"

var environmentsCache = make(map[string]*Environment)

func Environments(selector string, showSecrets, showMyTeams,
	showServiceAccount, showServiceAccountLoginToken, getCloudCredentials, jsonFormat bool) {

	envs, err := environmentsBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Environment(s): %v", err)
	}
	if len(envs) == 0 {
		if jsonFormat {
			log.Print("No Environments\n")
		} else {
			fmt.Print("No Environments\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(envs) == 1 {
				toMarshal = &envs[0]
			} else {
				toMarshal = envs
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Environments:\n")
			errors := make([]error, 0)
			for _, env := range envs {
				errors = formatEnvironmentEntity(&env, showSecrets, showMyTeams,
					showServiceAccount, showServiceAccountLoginToken, getCloudCredentials, errors)
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

func formatEnvironmentEntity(env *Environment, showSecrets, showMyTeams,
	showServiceAccount, showServiceAccountLoginToken, getCloudCredentials bool, errors []error) []error {

	title := fmt.Sprintf("%s [%s]", env.Name, env.Id)
	if env.Description != "" {
		title = fmt.Sprintf("%s - %s", title, env.Description)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(env.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(env.Tags, ", "))
	}
	if env.CloudAccount != "" {
		account, err := cloudAccountById(env.CloudAccount)
		if err != nil {
			errors = append(errors, err)
		} else {
			fmt.Printf("\t\tCloud Account: %s\n", formatCloudAccount(account))
		}
		if getCloudCredentials {
			keys, err := cloudAccountCredentials(account.Id, account.Kind)
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
	}
	resource := fmt.Sprintf("%s/%s", environmentsResource, env.Id)
	if len(env.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
		for _, param := range sortParameters(env.Parameters) {
			formatted, err := formatParameter(resource, param, showSecrets)
			fmt.Printf("\t\t%s\n", formatted)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(env.Providers) > 0 {
		fmt.Print("\t\tProviders:\n")
		for i, provider := range env.Providers {
			fmt.Printf("\t\t    %02d  %s [%s]\n", i, provider.Name, provider.Kind)
			provides := "(none)"
			if len(provider.Provides) > 0 {
				provides = strings.Join(provider.Provides, ", ")
			}
			fmt.Printf("\t\t\tProvides: %s\n", provides)
			if len(provider.Parameters) > 0 {
				fmt.Print("\t\t\tParameters:\n")
				for _, param := range sortParameters(provider.Parameters) {
					formatted, err := formatParameter(resource, param, showSecrets)
					fmt.Printf("\t\t\t%s\n", formatted)
					if err != nil {
						errors = append(errors, err)
					}
				}
			}
		}
	}
	if len(env.TeamsPermissions) > 0 {
		formatted := formatTeams(env.TeamsPermissions)
		fmt.Printf("\t\tTeams: %s\n", formatted)
		if showMyTeams {
			teams, err := myTeams(env.Id)
			formatted := formatTeams(teams)
			fmt.Printf("\t\tMy Teams: %s\n", formatted)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if showServiceAccount {
		teams, err := myTeams(env.Id)
		if err != nil {
			errors = append(errors, err)
		} else {
			if len(teams) > 0 {
				for _, team := range teams {
					account, err := serviceAccount(env.Id, team.Id)
					if err != nil {
						errors = append(errors, err)
					} else {
						formatted := formatServiceAccount(team, account, showServiceAccountLoginToken)
						fmt.Printf("\t\tService Account: %s\n", formatted)
					}
				}
			}
		}
	}
	return errors
}

func formatEnvironment(environment *Environment) {
	errors := formatEnvironmentEntity(environment, false, false, false, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func cachedEnvironmentBy(selector string) (*Environment, error) {
	env, cached := environmentsCache[selector]
	if !cached {
		var err error
		env, err = environmentBy(selector)
		if err != nil {
			return nil, err
		}
		environmentsCache[selector] = env
	}
	return env, nil
}

func environmentBy(selector string) (*Environment, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return environmentByName(selector)
	}
	return environmentById(selector)
}

func environmentsBy(selector string) ([]Environment, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return environmentsByName(selector)
	}
	environment, err := environmentById(selector)
	if err != nil {
		return nil, err
	}
	if environment != nil {
		return []Environment{*environment}, nil
	}
	return nil, nil
}

func environmentById(id string) (*Environment, error) {
	path := fmt.Sprintf("%s/%s", environmentsResource, url.PathEscape(id))
	var jsResp Environment
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Environments: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Environments, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func environmentByName(name string) (*Environment, error) {
	environments, err := environmentsByName(name)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Environment `%s`: %v", name, err)
	}
	if len(environments) == 0 {
		return nil, fmt.Errorf("No Environment `%s` found", name)
	}
	if len(environments) > 1 {
		return nil, fmt.Errorf("More than one Environment returned by name `%s`", name)
	}
	environment := environments[0]
	return &environment, nil
}

func environmentsByName(name string) ([]Environment, error) {
	path := environmentsResource
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var jsResp []Environment
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Environments: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Environments, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatEnvironmentRef(ref *EnvironmentRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func myTeams(environmentId string) ([]Team, error) {
	path := fmt.Sprintf("%s/%s/my-teams", environmentsResource, url.PathEscape(environmentId))
	var jsResp []Team
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Environment `%s` My Teams: %v",
			environmentId, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Environment `%s` My Teams, expected 200 HTTP",
			code, environmentId)
	}
	return jsResp, nil
}

func formatTeams(teams []Team) string {
	formatted := make([]string, 0, len(teams))
	for _, team := range teams {
		formatted = append(formatted, fmt.Sprintf("%s (%s)", team.Name, team.Role))
	}
	return strings.Join(formatted, ", ")
}

func serviceAccount(environmentId, teamId string) (*ServiceAccount, error) {
	path := fmt.Sprintf("%s/%s/service-account/%s", environmentsResource, url.PathEscape(environmentId), url.PathEscape(teamId))
	var jsResp ServiceAccount
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Team `%s` Service Account: %v",
			teamId, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Team `%s` Service Account, expected 200 HTTP",
			code, teamId)
	}
	return &jsResp, nil
}

func formatServiceAccount(team Team, account *ServiceAccount, showLoginToken bool) string {
	formatted := fmt.Sprintf("%s (%s) %s", team.Name, team.Role, account.Name)
	if showLoginToken {
		formatted = fmt.Sprintf("%s\n\t\t\tLogin Token: %s", formatted, account.LoginToken)
	}
	return formatted
}

func CreateEnvironment(name, cloudAccountSelector string) {
	cloudAccount, err := cloudAccountBy(cloudAccountSelector)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Environment: %v", err)
	}
	if cloudAccount == nil {
		log.Fatal("Unable to create SuperHub Environment: Cloud Account not found")
	}
	req := &EnvironmentRequest{
		Name:         name,
		CloudAccount: cloudAccount.Id,
		Parameters:   []Parameter{},
		Providers:    []Provider{},
	}
	environment, err := createEnvironment(req)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Environment: %v", err)
	}
	formatEnvironment(environment)
}

func createEnvironment(environment *EnvironmentRequest) (*Environment, error) {
	var jsResp Environment
	code, err := post(hubApi, environmentsResource, environment, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Environment, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func DeleteEnvironment(selector string) {
	err := deleteEnvironment(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Environment: %v", err)
	}
}

func deleteEnvironment(selector string) error {
	environment, err := environmentBy(selector)
	if err != nil {
		return err
	}
	if environment == nil {
		return error404
	}
	path := fmt.Sprintf("%s/%s", environmentsResource, url.PathEscape(environment.Id))
	code, err := delete(hubApi, path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Environments, expected [202, 204] HTTP", code)
	}
	return nil
}
