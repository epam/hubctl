package lifecycle

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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
	stderr, err := impl.StderrPipe()
	if err != nil {
		log.Fatalf("Unable to obtain stderr pipe: %v", err)
	}
	stdout, err := impl.StdoutPipe()
	if err != nil {
		log.Fatalf("Unable to obtain stdout pipe: %v", err)
	}

	args := ""
	if len(impl.Args) > 1 {
		args = fmt.Sprintf(" %v", impl.Args[1:])
	}
	implBlurb := fmt.Sprintf("%s%s (%s)", impl.Path, args, impl.Dir)

	os.Stdout.Sync()
	os.Stderr.Sync()

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	var stdoutWritter io.Writer = &stdoutBuffer
	var stderrWritter io.Writer = &stderrBuffer
	if pipeOutputInRealtime {
		stdoutWritter = io.MultiWriter(&stdoutBuffer, os.Stdout)
		stderrWritter = io.MultiWriter(&stderrBuffer, os.Stderr)
		fmt.Printf("--- %s\n", implBlurb)
	}

	stdoutComplete := goWait(func() { io.Copy(stdoutWritter, stdout) })
	stderrComplete := goWait(func() { io.Copy(stderrWritter, stderr) })
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

	if err == nil {
		err = impl.Wait()
	}
	if err != nil {
		err = fmt.Errorf("%s: %v", implBlurb, err)
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
}
