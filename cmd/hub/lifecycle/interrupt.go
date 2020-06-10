package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
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
					log.Printf("%s, Hub CLI exiting... Send ^C again to force exit", sig.String())
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
