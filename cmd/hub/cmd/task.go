package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
)

var taskCmd = &cobra.Command{
	Use:   "task <get | terminate> ...",
	Short: "Automation tasks management",
}

var taskGetCmd = &cobra.Command{
	Use:   "get [-e environment]",
	Short: "Show a list of automation tasks",
	Long:  `Show a list of all automation tasks, optionally filtered by environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return task(args)
	},
}

var taskTerminateCmd = &cobra.Command{
	Use:   "terminate <id>",
	Short: "Terminate automation task",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terminateTask(args)
	},
}

func task(args []string) error {
	if len(args) != 0 {
		return errors.New("Task command has no positional arguments")
	}

	api.Tasks(environmentSelector, jsonFormat)

	return nil
}

func terminateTask(args []string) error {
	if len(args) != 1 {
		return errors.New("Terminate task command has one mandatory argument - id of automation task")
	}

	api.TerminateTask(args[0])

	return nil
}

func init() {
	taskGetCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Environment name or Id")
	taskGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	taskCmd.AddCommand(taskGetCmd)
	taskCmd.AddCommand(taskTerminateCmd)
	apiCmd.AddCommand(taskCmd)
}
