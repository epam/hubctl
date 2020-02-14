package ext

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"hub/config"
	"hub/util"
)

const hubDir = ".hub"

func scriptPath(what string) (string, error) {
	script := "hub-" + what

	// TODO check $PATH after ./.hub/ ?
	path, err := exec.LookPath(script)
	if err == nil {
		return path, nil
	}

	searchDirs := []string{filepath.Join(".", hubDir)}
	home := os.Getenv("HOME")
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, hubDir))
	} else {
		if config.Verbose {
			util.Warn("Unable to lookup $HOME: no home directory set in OS environment")
		}
	}
	searchDirs = append(searchDirs, "/usr/local/share/hub", "/usr/share/hub")

	for _, dir := range searchDirs {
		path := filepath.Join(dir, script)
		_, err := os.Stat(path)
		if err != nil {
			if util.NoSuchFile(err) {
				continue
			}
			util.Warn("Unable to stat `%s`: %v", path, err)
		} else {
			// TODO check file mode
			// TODO allow extension placement in a dedicated subdirectory
			return path, nil
		}
	}

	return "", fmt.Errorf("Extension not found in $PATH, nor %v", searchDirs)
}

func RunExtension(what string, args []string) {
	code, err := runExtension(what, args)
	if err != nil {
		log.Fatalf("Unable to call `%s` extension: %v", what, err)
	}
	os.Exit(code)
}

func runExtension(what string, args []string) (int, error) {
	if config.Debug {
		log.Printf("Calling extension `%s` with args %v", what, args)
	}
	executable, err := scriptPath(what)
	if err != nil {
		return 0, err
	}
	if config.Debug {
		log.Printf("Found extension %s", executable)
	}
	cmd := exec.Cmd{
		Path:   executable,
		Args:   append([]string{filepath.Base(executable)}, args...),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), nil
		}
		code := 0
		if ps := cmd.ProcessState; ps != nil {
			code = ps.ExitCode()
		}
		return code, err
	}
	return 0, nil
}
