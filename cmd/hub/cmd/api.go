// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
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
	Short: "Invoke SuperHub API",
	Long: fmt.Sprintf(`Invoke arbitrary SuperHub API path, optionally sending JSON via stdin, for example:
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
	apiCmd.PersistentFlags().BoolVar(&config.ApiDerefSecrets, "deref-secrets",
		os.Getenv(envVarNameDerefSecrets) != "false",
		fmt.Sprintf("Always retrieve secrets to catch API errors (%s)", envVarNameDerefSecrets))
	apiCmd.PersistentFlags().IntVar(&config.ApiTimeout, "timeout", 30,
		"API HTTP timeout in seconds, HUB_API_TIMEOUT")
	apiCmd.AddCommand(apiInvokeCmd)
	RootCmd.AddCommand(apiCmd)
}
