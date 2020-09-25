package cmd

import (
	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/metrics"
)

func maybeMeterCommand(cmd *cobra.Command) {
	for cmd2 := cmd; cmd2 != nil; cmd2 = cmd2.Parent() {
		if ann := cmd.Annotations; ann != nil {
			if metering, exist := ann["usage-metering"]; exist {
				pipe := metrics.MeterCommand(cmd, metering == "tags")
				if cmdCtx := cmdContext(cmd); cmdCtx != nil {
					cmdCtx.Pipe = pipe
				}
				break
			}
		}
	}
}
