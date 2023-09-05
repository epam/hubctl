// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/ext"
	"github.com/epam/hubctl/cmd/hub/util"
)

var (
	knownExtensions = []string{"toolbox", "pull", "ls", "show", "configure", "stack"}
)

var extensionCmd = cobra.Command{
	Use:   "",
	Short: "`%s` extension",
	RunE: func(cmd *cobra.Command, args []string) error {
		newArgs := make([]string, 0, 1+len(args))
		newArgs = append(newArgs, cmd.Use)
		newArgs = append(newArgs, args...)
		return arbitraryExtension(newArgs)
	},
	DisableFlagParsing: true,
}

var arbitraryExtensionCmd = &cobra.Command{
	Use:   "ext [subcommands...]",
	Short: "Call arbitrary extension",
	Long:  "Call arbitrary extension via `hub-<extension name>` calling convention",
	RunE: func(cmd *cobra.Command, args []string) error {
		return arbitraryExtension(args)
	},
	DisableFlagParsing: true,
}

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Manage Hub CTL extensions",
}

var extensionsInstallCmd = &cobra.Command{
	Use:   "install [dir]",
	Short: "Install Hub CTL extensions",
	Long: `Install Hub CTL extension into ~/.hub/ by cloning git@github.com:epam/hub-extensions.git
and installing dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return extensionsInstall(args)
	},
}

var extensionsUpdateCmd = &cobra.Command{
	Use:   "update [dir]",
	Short: "Update Hub CTL extensions",
	Long: `Update Hub CTL extension via hub pull in ~/.hub/
and refreshing dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return extensionsUpdate(args)
	},
}

func extension(what []string, args []string) error {
	config.AggWarnings = false
	if hub := os.Getenv(envVarNameHubCli); hub == "" {
		if bin, err := os.Executable(); err == nil {
			os.Setenv(envVarNameHubCli, bin)
		} else {
			util.Warn("Unable to determine path to Hub CTL executable - `hub <extension>` might be broken: %v", err)
		}
	}
	ext.RunExtension(what, args)
	return nil
}

func arbitraryExtension(args []string) error {
	stopWhat := false
	what := make([]string, 0, 1)
	finalArgs := make([]string, 0, len(args))
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if !stopWhat {
				what = append(what, arg)
			} else {
				finalArgs = append(finalArgs, args[i:]...)
				break
			}
		} else {
			finalArgs = append(finalArgs, arg)
			if len(what) > 0 {
				stopWhat = true
			}
		}
	}
	if len(what) == 0 {
		return errors.New("Extensions command has at least one mandatory argument - the name of extension command to call")
	}

	return extension(what, finalArgs)
}

func extensionsInstall(args []string) error {
	if len(args) != 0 && len(args) != 1 {
		return errors.New("Extensions Install command has one optional argument - path to Hub CTL extensions folder")
	}
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	config.AggWarnings = false
	ext.Install(dir)
	return nil
}

func extensionsUpdate(args []string) error {
	if len(args) != 0 && len(args) != 1 {
		return errors.New("Extensions Update command has one optional argument - path to Hub CTL extensions folder")
	}
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	config.AggWarnings = false
	ext.Update(dir)
	return nil
}

func init() {
	for _, e := range knownExtensions {
		cmd := extensionCmd
		cmd.Use = e
		cmd.Short = fmt.Sprintf(cmd.Short, e)
		RootCmd.AddCommand(&cmd)
	}
	RootCmd.AddCommand(arbitraryExtensionCmd)
	extensionsCmd.AddCommand(extensionsInstallCmd)
	extensionsCmd.AddCommand(extensionsUpdateCmd)
	RootCmd.AddCommand(extensionsCmd)
}
