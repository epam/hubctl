package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"hub/api"
)

var (
	instanceShowLogs bool
	kubeconfigOutput string
)

var instanceCmd = &cobra.Command{
	Use:   "instance <get | create | delete> ...",
	Short: "Create and manage Stack Instances",
}

var instanceGetCmd = &cobra.Command{
	Use:   "get [id | domain]",
	Short: "Show a list of instances or details about the instance",
	Long: `Show a list of all user accessible stack instances or details about
the particular instance (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return instance(args)
	},
}

var instanceCreateCmd = &cobra.Command{
	Use:   "create < instance.json",
	Short: "Create Stack Instance",
	Long:  `Create Stack Instance by sending JSON via stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createInstance(args)
	},
}

var instanceDeployCmd = &cobra.Command{
	Use:   "deploy <id | domain>",
	Short: "Deploy Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deployInstance(args)
	},
}

var instanceUndeployCmd = &cobra.Command{
	Use:   "undeploy <id | domain>",
	Short: "Undeploy Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return undeployInstance(args)
	},
}

var instanceDeleteCmd = &cobra.Command{
	Use:   "delete <id | domain>",
	Short: "Delete Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteInstance(args)
	},
}

var instanceKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig <id | domain>",
	Short: "Download Stack Instance Kubeconfig by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return kubeconfigInstance(args)
	},
}

func instance(args []string) error {
	if len(args) > 1 {
		return errors.New("Instance command has one optional argument - id or domain of the Stack Instance")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.StackInstances(selector, showSecrets, instanceShowLogs)

	return nil
}

func createInstance(args []string) error {
	if len(args) > 0 {
		return errors.New("Create Instance command has no arguments")
	}

	api.CreateStackInstance(os.Stdin)

	return nil
}

func deployInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Deploy Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.DeployStackInstance(args[0], waitAndTailDeployLogs)

	return nil
}

func undeployInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Undeploy Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.UndeployStackInstance(args[0], waitAndTailDeployLogs)

	return nil
}

func deleteInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.DeleteStackInstance(args[0])

	return nil
}

func kubeconfigInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Kubeconfig Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.KubeconfigStackInstance(args[0], kubeconfigOutput)

	return nil
}

func init() {
	instanceGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "s", false,
		"Show secrets")
	instanceGetCmd.Flags().BoolVarP(&instanceShowLogs, "logs", "l", false,
		"Show logs")
	instanceDeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceUndeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceKubeconfigCmd.Flags().StringVarP(&kubeconfigOutput, "output", "o", "",
		"Set output filename (default to kubeconfig-DOMAIN.yaml)")
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceCreateCmd)
	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceUndeployCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceKubeconfigCmd)
	apiCmd.AddCommand(instanceCmd)
}
