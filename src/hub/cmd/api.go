package cmd

import (
	"fmt"
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
		os.Getenv(envVarNameDerefSecrets) != "false",
		fmt.Sprintf("Always retrieve secrets to catch API errors (%s)", envVarNameDerefSecrets))
	apiCmd.PersistentFlags().IntVar(&config.ApiTimeout, "timeout", 30,
		"API HTTP timeout in seconds")
	RootCmd.AddCommand(apiCmd)
}
