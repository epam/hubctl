// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/cel"
	"github.com/agilestacks/hub/cmd/hub/config"
)

var (
	celAutoVars  bool
	celYamlValue bool
)

var celCmd = &cobra.Command{
	Use:   "cel [-a] [-y] <expression> [bind.some.name=value1,...]",
	Short: "Evaluate CEL expression",
	Long: `Evaluate CEL expression.
https://github.com/google/cel-spec/blob/master/doc/langdef.md

Set -d / --debug to print CEL internals.

$ hub cel '{"aws": "gp2", "gcp": "pd-ssd"}[cloud.kind]' cloud.kind=gcp
pd-ssd

$ hub cel -y '#{3 - int({"prime": "7"}[prime])}-${q}' prime=prime,q=x
-4-x
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return celEval(args)
	},
}

func celEval(args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return errors.New("CEL command has only one mandatory argument - CEL expressiom, and additional optional argument - variable bindings")
	}
	expression := args[0]
	bindings := ""
	if len(args) == 2 {
		bindings = args[1]
	}

	config.AggWarnings = false
	cel.Eval(expression, bindings, celAutoVars, celYamlValue)

	return nil
}

func init() {
	celCmd.Flags().BoolVarP(&celAutoVars, "auto-vars", "a", false,
		"Auto-resolve variable into \"<variable name>\" if not found in bindings")
	celCmd.Flags().BoolVarP(&celYamlValue, "yaml-value-expression", "y", false,
		"Process as #{CEL expression} YAML manifest value: expression")
	RootCmd.AddCommand(celCmd)
}
