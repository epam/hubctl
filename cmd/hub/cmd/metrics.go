package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/metrics"
)

var commandsToMeter = []*cobra.Command{
	apiCmd,
	elaborateCmd,
	deployCmd,
	undeployCmd,
	backupCreateCmd,
}

func maybeMeterCommand(cmd *cobra.Command) {
	for _, toMeter := range commandsToMeter {
		for cmd2 := cmd; cmd2 != nil; cmd2 = cmd2.Parent() {
			if cmd2 == toMeter {
				metrics.MeterCommand(cmd)
				break
			}
		}
	}
}
