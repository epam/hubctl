package cmd

import (
	"github.com/spf13/cobra"

	"hub/api"
)

var (
	logsExitOnCompletedOperation bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [entity kind/][id | name | domain ...]",
	Short: "Tail logs and status updates",
	Long: `Tail deployment logs, lifecycle operation phases, and stack instance status changes.
A filtering list of Ids or domain names may be supplied to limit the output, otherwise
everything accessible to the current user is shown.

Entity kind is one of cloudAccount, environment, stackTemplate, stackInstance (default).

When --exit-on-completed-operation / -w is specified, then the command will tail logs
until all specified entities completes lifecycle operation, then CLI will exit.
Otherwise it will tail logs indefinitely until interrupted (the default).

If no filter is specified then the command will wait for one lifecycle operation
completion on any entity.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return logs(args)
	},
}

func logs(args []string) error {
	selectors := args
	api.Logs(selectors, logsExitOnCompletedOperation)

	return nil
}

func init() {
	logsCmd.Flags().BoolVarP(&logsExitOnCompletedOperation, "exit-on-completed-operation", "w", false,
		"Exit after current lifecycle operation completes (with success or failure)")
	apiCmd.AddCommand(logsCmd)
}
