// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

var interruptSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func watchInterrupt() context.Context {
	ctx, interrupted := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	unwatch := make(chan struct{})
	signal.Notify(sigs, interruptSignals...)
	go func() {
		for {
			select {
			case sig := <-sigs:
				if ctx.Err() != nil {
					os.Exit(3)
				}
				interrupted()
				if config.Verbose {
					log.Writer().Write([]byte("\n"))
					log.Printf("%s, Hub CTL exiting... Send ^C again to force exit", sig.String())
				}

			case <-unwatch:
				signal.Reset(interruptSignals...)
				return
			}
		}
	}()
	util.AtDone(func() <-chan struct{} {
		unwatch <- struct{}{}
		return nil
	})
	return ctx
}
