// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var secretCmd = cobra.Command{
	Use:   "secret <get | create> ...",
	Short: "Manage %s secrets",
}

var getSecretCmd = cobra.Command{
	Use:   "get <selector> <secret uuid>",
	Short: "Get secret parameter value",
}

var createSecretCmd = cobra.Command{
	Use:   "create <selector> <secret name>[|component] <secret kind> <value | key:value | - ...>",
	Short: "Create secret parameter",
	Long: `To create Secret, provide:

- selector is either environment name or id, template name or id, instance full domain name or id
- secret name
- secret kind, one of: password cloudAccount cloudAccessKeys privateKey
	certificate sshKey usernamePassword text license
	token bearerToken accessToken refreshToken loginToken
- secret plain value, or a number of key:value pairs appropriate for particular secret kind, ie.:
	password: password
	usernamePassword: username, password
	text: text
	cloudAccount: cloud, roleArn, externalId, duration
	cloudAccessKeys: cloud, accessKey, secretKey
	privateKey: privateKey
	certificate: certificate
	sshKey: sshKey
	license: licenseKey
	*token: *token
- if secret "value" is "-" then it is read from stdin`,
}

func getSecret(entityKind string, args []string) error {
	if len(args) != 2 {
		return errors.New("Get Secret command has two mandatory arguments - entity selector and secret UUID")
	}

	selector := args[0]
	uuid := args[1]

	api.GetSecret(entityKind, selector, uuid, jsonFormat)

	return nil
}

func createSecret(entityKind string, args []string) error {
	if len(args) < 4 {
		return errors.New("Create Secret command has four of more arguments")
	}

	selector := args[0]
	supportedKinds := []string{"environment", "template", "instance"}
	if !util.Contains(supportedKinds, entityKind) {
		return fmt.Errorf("Bad entity kind `%s`; supported %v", entityKind, supportedKinds)
	}

	qName := args[1]
	kind := args[2]
	values := make(map[string]string)
	if len(args) == 4 { // probably plain value prefixed with `key:`
		value := args[3]
		maybeKey := kind + ":"
		if strings.HasPrefix(value, maybeKey) && len(value) > len(maybeKey) {
			value = value[len(maybeKey)+1:]
		}
		if value == "-" {
			valueBytes, err := ioutil.ReadAll(os.Stdin)
			if err != nil || len(valueBytes) == 0 {
				return fmt.Errorf("Bad secret value read from stdin (read %d bytes): %s",
					len(valueBytes), util.Errors2(err))
			}
			value = string(valueBytes)
		}
		values[kind] = value
	} else {
		// must be `key:value` format
		for _, kv := range args[3:] {
			i := strings.Index(kv, ":")
			if i <= 0 {
				return fmt.Errorf("Bad key:value format `%s`", kv)
			}
			key := kv[:i]
			value := ""
			if i < len(kv)-1 {
				value = kv[i+1:]
			}
			values[key] = value
		}
	}

	name, component := util.SplitQName(qName)
	api.CreateSecret(entityKind, selector, name, component, kind, values)

	return nil
}

func init() {
	parents := map[string]*cobra.Command{
		"cloudaccount": cloudAccountCmd,
		"environment":  environmentCmd,
		"template":     templateCmd,
		"instance":     instanceCmd,
		"application":  applicationCmd,
	}

	for entityKind, entityCmd := range parents {
		k := entityKind
		getCmd := &cobra.Command{}
		*getCmd = getSecretCmd
		getCmd.RunE = func(cmd *cobra.Command, args []string) error { return getSecret(k, args) }
		getCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
			"JSON output")

		createCmd := &cobra.Command{}
		*createCmd = createSecretCmd
		createCmd.RunE = func(cmd *cobra.Command, args []string) error { return createSecret(k, args) }

		parentCmd := &cobra.Command{}
		*parentCmd = secretCmd
		parentCmd.Short = fmt.Sprintf(secretCmd.Short, entityKind)

		if entityKind != "cloudaccount" && entityKind != "application" {
			parentCmd.AddCommand(createCmd)
		}
		parentCmd.AddCommand(getCmd)
		entityCmd.AddCommand(parentCmd)
	}
}
