// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/state"
	"github.com/epam/hubctl/cmd/hub/util"
)

var (
	explainGlobal bool
	explainRaw    bool
	explainOpLog  bool
	explainInKv   bool
	explainInSh   bool
	explainInJson bool
	explainInYaml bool
	explainColor  bool
)

var explainCmd = &cobra.Command{
	Use:   "explain [hub.yaml.elaborate] hub.yaml.state[,s3://bucket/hub.yaml.state]",
	Short: "Explain stack outputs, provides, and parameters",
	Long: `Display stack outputs, component's parameters, outputs, and capabilities.
Parameters and outputs are read from state file. Elaborate file is optional.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return explain(args)
	},
}

func explain(args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return errors.New("Explain command has two arguments - path to Stack Elaborate file (optional) and to State file")
	}

	elaborateManifests := []string{}
	i := 0
	if len(args) == 2 {
		elaborateManifests = util.SplitPaths(args[0])
		i = 1
	}
	stateManifests := util.SplitPaths(args[i])

	format := "text"
	if explainInKv {
		format = "kv"
	} else if explainInSh {
		format = "sh"
	} else if explainInJson {
		format = "json"
	} else if explainInYaml {
		format = "yaml"
	}

	state.Explain(elaborateManifests, stateManifests, explainOpLog, explainGlobal, componentName, explainRaw, format, explainColor)

	return nil
}

func init() {
	explainCmd.Flags().BoolVarP(&explainGlobal, "global", "g", false,
		"Display Stack or Application parameters and outputs")
	explainCmd.Flags().StringVarP(&componentName, "component", "c", "",
		"Component to explain")
	explainCmd.Flags().BoolVarP(&explainRaw, "raw-outputs", "r", false,
		"Display raw component outputs")
	explainCmd.Flags().BoolVarP(&explainOpLog, "op-log", "l", false,
		"Display operations log (only)")
	explainCmd.Flags().BoolVarP(&explainInKv, "kv", "", false,
		"key=value output")
	explainCmd.Flags().BoolVarP(&explainInSh, "sh", "", false,
		"Shell output")
	explainCmd.Flags().BoolVarP(&explainInJson, "json", "", false,
		"JSON output")
	explainCmd.Flags().BoolVarP(&explainInYaml, "yaml", "", false,
		"YAML output")
	explainCmd.Flags().BoolVarP(&explainColor, "color", "", isatty.IsTerminal(os.Stdout.Fd()),
		"Colorized output")
	RootCmd.AddCommand(explainCmd)
}
