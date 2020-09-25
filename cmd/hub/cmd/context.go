package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/util"
)

type ContextKey string

type CmdContext struct {
	Pipe io.WriteCloser
}

var contextKey = ContextKey("cmd")

func cmdContext(cmd *cobra.Command) *CmdContext {
	if ctx := cmd.Context(); ctx != nil {
		if cmdCtx, ok := ctx.Value(contextKey).(*CmdContext); ok && cmdCtx != nil {
			return cmdCtx
		}
	}
	util.Warn("No command context detected")
	return nil
}

func cmdContextPipe(cmd *cobra.Command) io.WriteCloser {
	cmdCtx := cmdContext(cmd)
	if cmdCtx != nil {
		return cmdCtx.Pipe
	}
	return nil
}
