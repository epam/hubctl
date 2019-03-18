package util

import (
	"fmt"
	"log"
	"os"

	"hub/config"
)

var atDone []func() <-chan struct{}

func MaybeFatalf(format string, v ...interface{}) {
	if config.Force {
		Warn(format, v...)
	} else {
		Done()
		log.Fatalf(format, v...)
	}
}

func MaybeFatalf2(cleanup func(string, bool), format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if config.Force {
		Warn("%s", msg)
	} else {
		log.Print(msg)
	}
	if cleanup != nil {
		cleanup(msg, !config.Force)
	}
	if !config.Force {
		Done()
		os.Exit(1)
	}
}

func AtDone(cleanup func() <-chan struct{}) {
	atDone = append(atDone, cleanup)
}

func Done() {
	var chs []<-chan struct{}
	for _, cleanup := range atDone {
		ch := cleanup()
		if ch != nil {
			chs = append(chs, ch)
		}
	}
	for _, ch := range chs {
		<-ch
	}
}
