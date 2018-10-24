package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hub/api"
	"hub/util"
)

var secretCmd = &cobra.Command{
	Use:   "secret [entity kind/]<selector> <secret name> <secret kind> <value | key:value | - ...>",
	Short: "Create secret in Environment, Template, or Stack Instance",
	Long: `To create Secret, provide:

- optionally, entity kind is one of: environment (default), stackTemplate, stackInstance
- selector is either environment name or id, template name or id, instance full domain name or id
- secret name
- secret kind, one of: password, usernamePassword, text, cloudAccessKeys, privateKey, certificate, sshKey, license
- secret plain value, or a number of key:value pairs appropriate for particular secret kind, ie.:
	password: password
	usernamePassword: username, password
	text: text
	cloudAccessKeys: accessKey, secretKey
	privateKey: privateKey
	certificate: certificate
	sshKey: sshKey
	license: license
- of secret "value" is "-" then it is read from stdin`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return secret(args)
	},
}

func secret(args []string) error {
	if len(args) < 4 {
		return errors.New("Secret command has four of more arguments")
	}

	entityKind := "environment"
	selector := args[0]
	if spec := strings.SplitN(selector, "/", 2); len(spec) == 2 {
		entityKind = spec[0]
		selector = spec[1]
	}
	supportedKinds := []string{"environment", "stackTemplate", "stackInstance"}
	if !util.Contains(supportedKinds, entityKind) {
		return fmt.Errorf("Bad entity kind `%s`; supported %v", entityKind, supportedKinds)
	}

	name := args[1]
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
	api.CreateSecret(entityKind, selector, name, kind, values)

	return nil
}

func init() {
	apiCmd.AddCommand(secretCmd)
}
