package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"hub/api"
)

var applicationCmd = &cobra.Command{
	Use:   "application <get | create | delete> ...",
	Short: "Create and manage Applications",
}

var applicationGetCmd = &cobra.Command{
	Use:   "get [id | domain]",
	Short: "Show a list of applications or details about the application",
	Long: `Show a list of all user accessible applications or details about
the particular application (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return application(args)
	},
}

func application(args []string) error {
	if len(args) > 1 {
		return errors.New("Application command has one optional argument - id or domain of the application")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Applications(selector)

	return nil
}

func init() {
	applicationCmd.AddCommand(applicationGetCmd)
	apiCmd.AddCommand(applicationCmd)
}
