package cmd

import (
	"github.com/spf13/cobra"

	"hub/api"
)

var logsCmd = &cobra.Command{
	Use:   "logs [entity kind/][id | name | domain ...]",
	Short: "Tail logs and status updates",
	Long: `Tail deployment logs, lifecycle operation phases, and stack instance status changes.
A list of Ids or domain names may be supplied to limit the output,
otherwise everything accessibly to the current user is shown.
Entity kind is one of cloudAccount, environment, stackTemplate, stackInstance (default)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return logs(args)
	},
}

func logs(args []string) error {
	selectors := args
	api.Logs(selectors)

	return nil
}

func init() {
	apiCmd.AddCommand(logsCmd)
}
