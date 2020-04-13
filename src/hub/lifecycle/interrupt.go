package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"

	"hub/config"
	"hub/util"
)

func watchInterrupt() context.Context {
	ctx, interrupted := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	unwatch := make(chan struct{})
	signal.Notify(sig, os.Interrupt)
	go func() {
		for {
			select {
			case <-sig:
				if ctx.Err() != nil {
					os.Exit(3)
				}
				interrupted()
				if config.Verbose {
					log.Writer().Write([]byte("\n"))
					log.Printf("Hub CLI exiting... Send ^C again to force exit")
				}

			case <-unwatch:
				signal.Reset(os.Interrupt)
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
