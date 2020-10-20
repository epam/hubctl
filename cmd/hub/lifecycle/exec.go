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

func execImplementation(impl *exec.Cmd, pipeOutputInRealtime bool) ([]byte, []byte, error) {
	stderrImpl, err := impl.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to obtain sub-process stderr pipe: %v", err)
	}
	stdoutImpl, err := impl.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to obtain sub-process stdout pipe: %v", err)
	}

	args := ""
	if len(impl.Args) > 1 {
		args = fmt.Sprintf(" %v", impl.Args[1:])
	}
	implBlurb := fmt.Sprintf("%s%s (%s)", impl.Path, args, impl.Dir)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	var stdoutWritter io.Writer = &stdoutBuffer
	var stderrWritter io.Writer = &stderrBuffer
	logOutput := log.Writer()
	if pipeOutputInRealtime {
		var stdout io.Writer = os.Stdout
		var stderr io.Writer = os.Stderr

		if config.Tty && !config.Debug {
			stdoutTerminal := isatty.IsTerminal(os.Stdout.Fd())
			stderrTerminal := isatty.IsTerminal(os.Stderr.Fd())
			to := os.Stdout
			if !stdoutTerminal && stderrTerminal {
				to = os.Stderr
			}
			tail := newTail(to)
			if stdoutTerminal || config.TtyForced {
				stdout = tail
			}
			if stderrTerminal || config.TtyForced {
				stderr = tail
			}
			// send CLI messages to common stream so that output is formatted correctly
			log.SetOutput(tail)
		}

		stdoutWritter = io.MultiWriter(&stdoutBuffer, stdout)
		stderrWritter = io.MultiWriter(&stderrBuffer, stderr)
		fmt.Printf("--- %s\n", implBlurb)
	}

	os.Stdout.Sync()
	os.Stderr.Sync()

	stdoutComplete := goWait(func() { io.Copy(stdoutWritter, stdoutImpl) })
	stderrComplete := goWait(func() { io.Copy(stderrWritter, stderrImpl) })
	// Wait will close the pipe after seeing the command exit, so most callers
	// need not close the pipe themselves; however, an implication is that it is
	// incorrect to call Wait before all reads from the pipe have completed.
	// For the same reason, it is incorrect to call Run when using StdoutPipe.
	err = impl.Start()
	<-stdoutComplete
	<-stderrComplete

	if pipeOutputInRealtime {
		fmt.Print("---\n")
	}

	os.Stdout.Sync()
	os.Stderr.Sync()

	log.SetOutput(logOutput)

	if err == nil {
		err = impl.Wait()
	}
	if err != nil {
		err = fmt.Errorf("%s: %v", implBlurb, err)
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
}
