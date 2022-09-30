// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/util"
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
