// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/util"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Hub CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Hub CLI %s %s\n", util.Version(), runtime.Version())
	},
}
