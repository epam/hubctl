package cmd

import (
	"github.com/spf13/cobra"

	"hub/lifecycle"
)

var undeployCmd = &cobra.Command{
	Use:   "undeploy hub.yaml.elaborate",
	Short: "Undeploy stack",
	Long:  `Undeploy stack instance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return undeploy(args)
	},
}

func undeploy(args []string) error {
	request, err := lifecycleRequest(args, "undeploy")
	if err != nil {
		return err
	}
	lifecycle.Execute(request)
	return nil
}

func init() {
	initDeployUndeployFlags(undeployCmd, "undeploy")
	undeployCmd.Flags().BoolVarP(&guessComponent, "guess", "", true,
		"Guess component to start undeploy with (useful for failed deployments)")
	undeployCmd.Flags().StringVarP(&limitComponent, "limit", "l", "",
		"Component to stop undeploy at")
	RootCmd.AddCommand(undeployCmd)
}
