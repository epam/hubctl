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
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/epam/hubctl/cmd/hub/api"
	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

var apiCmd = &cobra.Command{
	Use:   "api ...",
	Short: "API to access SuperHub.io",
	Annotations: map[string]string{
		"usage-metering": "",
	},
}

var apiInvokeCmd = &cobra.Command{
	Use:   "invoke <METHOD> <path> [< request.json]",
	Short: "Invoke HubCTL API",
	Long: fmt.Sprintf(`Invoke arbitrary HubCTL API path, optionally sending JSON via stdin, for example:
%[1]s
	{
	}
%[1]s

Request is sent with Authorization header`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return apiInvoke(args)
	},
}

func apiInvoke(args []string) error {
	if len(args) != 2 {
		return errors.New("Invoke API command has two mandatory arguments - HTTP method and resource path")
	}

	method := args[0]
	path := args[1]
	var body io.Reader
	if util.Contains(api.MethodsWithJsonBody, method) {
		body = os.Stdin
	}
	api.Invoke(method, path, body)

	return nil
}

func init() {
	onInitialize(func() {
		if api := viper.GetString("api"); api != "" {
			config.ApiBaseUrl = api
		}
		if loginToken := viper.GetString("token"); loginToken != "" {
			config.ApiLoginToken = loginToken
		}
		if t := viper.GetString("api-timeout"); t != "" {
			if timeout, err := strconv.Atoi(t); err == nil && timeout > 0 {
				config.ApiTimeout = timeout
			}
		}
	})

	apiDefault := os.Getenv(envVarNameHubApi)
	if apiDefault == "" {
		apiDefault = "https://api.agilestacks.io"
	}
	apiCmd.PersistentFlags().StringVar(&config.ApiBaseUrl, "api", apiDefault, "Hub API service base URL, HUB_API")
	apiCmd.PersistentFlags().BoolVar(&config.ApiDerefSecrets, "deref-secrets",
		os.Getenv(envVarNameDerefSecrets) != "false",
		fmt.Sprintf("Always retrieve secrets to catch API errors (%s)", envVarNameDerefSecrets))
	apiCmd.PersistentFlags().IntVar(&config.ApiTimeout, "timeout", 30,
		"API HTTP timeout in seconds, HUB_API_TIMEOUT")
	apiCmd.AddCommand(apiInvokeCmd)
	RootCmd.AddCommand(apiCmd)
}
