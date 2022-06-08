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
