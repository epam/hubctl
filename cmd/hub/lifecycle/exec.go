// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/mattn/go-isatty"

	"github.com/agilestacks/hub/cmd/hub/config"
)

func goWait(routine func()) chan string {
	ch := make(chan string)
	wrapper := func() {
		routine()
		ch <- "done"
	}
	go wrapper()
	return ch
}

func execImplementation(impl *exec.Cmd, passStdin, paginate bool) ([]byte, []byte, error) {
	stderrImpl, err := impl.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to obtain sub-process stderr pipe: %v", err)
	}
	stdoutImpl, err := impl.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to obtain sub-process stdout pipe: %v", err)
	}

	logOutput := log.Writer()

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr

	if paginate && config.Tty && !config.Debug {
		stdoutTerminal := isatty.IsTerminal(os.Stdout.Fd())
		stderrTerminal := isatty.IsTerminal(os.Stderr.Fd())
		to := os.Stdout
		if !stdoutTerminal && stderrTerminal {
			to = os.Stderr
		}
		tail := newTail(to)
		defer tail.Close()
		if stdoutTerminal || config.TtyForced {
			stdout = tail
		}
		if stderrTerminal || config.TtyForced {
			stderr = tail
		}
		// send CLI messages to common stream so that output is formatted correctly
		log.SetOutput(tail)
	}

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	stdoutWritter := io.MultiWriter(&stdoutBuffer, stdout)
	stderrWritter := io.MultiWriter(&stderrBuffer, stderr)
	if impl.Path != "" {
		dir := impl.Dir
		fmt.Printf("--- Dir: %s\n", dir)
		fmt.Printf("--- File: %s\n", impl.Path)
		args := ""
		if len(impl.Args) > 1 {
			args = fmt.Sprintf("Args: %v", impl.Args[1:])
		}
		if args != "" {
			fmt.Printf("--- %s\n", args)
		}
	}
	os.Stdout.Sync()
	os.Stderr.Sync()

	if passStdin {
		impl.Stdin = os.Stdin
	}

	stdoutComplete := goWait(func() { io.Copy(stdoutWritter, stdoutImpl) })
	stderrComplete := goWait(func() { io.Copy(stderrWritter, stderrImpl) })
	// Wait will close the pipe after seeing the command exit, so most callers
	// need not close the pipe themselves; however, an implication is that it is
	// incorrect to call Wait before all reads from the pipe have completed.
	// For the same reason, it is incorrect to call Run when using StdoutPipe.
	err = impl.Start()
	<-stdoutComplete
	<-stderrComplete

	if impl.Path != "" {
		fmt.Printf("--- \n")
	}
	os.Stdout.Sync()
	os.Stderr.Sync()

	log.SetOutput(logOutput)

	if err == nil {
		err = impl.Wait()
	}
	if err != nil {
		err = fmt.Errorf("%v", err)
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
}
