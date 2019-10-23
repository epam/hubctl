package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"hub/config"
)

var apiCmd = &cobra.Command{
	Use:   "api ...",
	Short: "Use remote Automation Hub API to access SuperHub",
}

func init() {
	apiCmd.PersistentFlags().BoolVar(&config.ApiDerefSecrets, "deref-secrets",
		os.Getenv(envVarNameDerefSecrets) != "false", "Always retrieve secrets to catch API errors")
	RootCmd.AddCommand(apiCmd)
}
