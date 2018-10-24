package cmd

import (
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api ...",
	Short: "Use remote Automation Hub API - Control Plane",
}

func init() {
	RootCmd.AddCommand(apiCmd)
}
