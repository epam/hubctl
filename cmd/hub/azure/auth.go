// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package azure

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	storageManagement "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-11-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

const storageKeyHelp = "Please set AZURE_STORAGE_KEY environment variable, or AZURE_SUBSCRIPTION_ID, AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET"

var (
	defaultSettings    map[string]string
	defaultEnvironment *azure.Environment
)

func environment(values map[string]string) (*azure.Environment, error) {
	if name, exist := values[auth.EnvironmentName]; !exist || name == "" {
		return &azure.PublicCloud, nil
	} else {
		env, err := azure.EnvironmentFromName(name)
		if err != nil {
			return nil, err
		}
		return &env, nil
	}
}

func settings() (map[string]string, *azure.Environment, error) {
	if defaultSettings != nil && defaultEnvironment != nil {
		return defaultSettings, defaultEnvironment, nil
	}
	var values map[string]string
	var env *azure.Environment
	set, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		file, err2 := auth.GetSettingsFromFile()
		if err2 != nil {
			return nil, nil, fmt.Errorf("Errors retrieving Azure settings: %v", util.Errors2(err, err2))
		}
		values = file.Values
		env, err = environment(values)
		if err != nil {
			return nil, nil, err
		}
	} else {
		values = set.Values
		_env := set.Environment
		env = &_env
	}
	if config.Trace {
		log.Printf("Azure settings:\n\t%v", values)
	}
	defaultSettings = values
	defaultEnvironment = env
	return values, env, nil
}

func keyvaultResource(env *azure.Environment) string {
	return env.KeyVaultEndpoint
}

func serviceManagementResource(env *azure.Environment) string {
	return env.ServiceManagementEndpoint
}

func authorizer(resourcePick func(env *azure.Environment) string) (autorest.Authorizer, error) {
	var err error
	var errs []error
	var authz autorest.Authorizer

	resource := resourcePick(&azure.PublicCloud)
	_, env, err := settings()
	if err != nil && env != nil {
		resource = resourcePick(env)
	}
	resource = strings.TrimSuffix(resource, "/")

	if authLocation := os.Getenv("AZURE_AUTH_LOCATION"); config.AzureCredentialsFile != "" || authLocation != "" {
		if config.AzureCredentialsFile != "" {
			os.Setenv("AZURE_AUTH_LOCATION", config.AzureCredentialsFile)
		}
		authz, err = auth.NewAuthorizerFromFile(resource)
		if err != nil {
			errs = append(errs, err)
		}
	} else {
		authz, err = auth.NewAuthorizerFromEnvironmentWithResource(resource)
		if err != nil {
			errs = append(errs, err)
			authz, err = auth.NewAuthorizerFromCLIWithResource(resource)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		err = fmt.Errorf("Unable to create Azure authorizer: %v", util.Errors2(errs...))
	} else {
		err = nil
	}
	if err != nil && authz != nil && config.Debug {
		log.Printf("Errors encountered while creating Azure authorizer: %v", err)
	}
	return authz, err
}

func storageManagementClient() (*storageManagement.AccountsClient, error) {
	authz, err := authorizer(serviceManagementResource)
	if err != nil && authz == nil {
		return nil, err
	}
	sets, _, _ := settings()
	subscriptionId := sets[auth.SubscriptionID]
	client := storageManagement.NewAccountsClient(subscriptionId)
	client.Authorizer = authz
	return &client, nil
}

func resourceGroup() (string, error) {
	osEnvVar := "AZURE_RESOURCE_GROUP_NAME"
	name := os.Getenv(osEnvVar)
	if name != "" {
		return name, nil
	}
	hardcoded := "hubctl"
	if config.Verbose {
		util.WarnOnce("Using hardcoded `%s` Azure resource group; set %s to override",
			hardcoded, osEnvVar)
	}
	return hardcoded, nil
}

func storageKeyFromApi(account string) (string, error) {
	mgmt, err := storageManagementClient()
	if err != nil {
		return "", err
	}
	resourceGroupName, err := resourceGroup()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), storageTimeout)
	defer cancel()
	resp, err := mgmt.ListKeys(ctx, resourceGroupName, account)
	if err != nil {
		return "", fmt.Errorf("Error listing storage account `%s` access keys: %v;\n\t%s",
			account, err, storageKeyHelp)
	}
	if keys := resp.Keys; keys != nil {
		if len(*keys) > 0 {
			for _, keyEntry := range *keys {
				if key := keyEntry.Value; key != nil && *key != "" {
					return *key, nil
				}
			}
		}
	}
	return "", fmt.Errorf("No storage account `%s` access keys found;\n\t%s", account, storageKeyHelp)
}

func storageKey(account string) (string, error) {
	vars := []string{"AZURE_STORAGE_ACCESS_KEY", "AZURE_STORAGE_KEY", "ARM_ACCESS_KEY"}
	for _, v := range vars {
		key := os.Getenv(v)
		if key != "" {
			return key, nil
		}
	}
	return storageKeyFromApi(account)
}
