// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ext

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

const hubDir = ".hub"

func ExtensionPath(what, args []string) (string, []string, error) {

	searchDirs := []string{filepath.Join(".", hubDir)}

	customHubDir := os.Getenv("HUB_EXTENSIONS")
	if customHubDir != "" {
		searchDirs = append(searchDirs, customHubDir)
	}

	home := os.Getenv("HOME")
	homeHubDir := ""
	if home != "" {
		homeHubDir = filepath.Join(home, hubDir)
		searchDirs = append(searchDirs, homeHubDir)
	} else {
		if config.Verbose {
			util.Warn("Unable to lookup $HOME: no home directory set in OS environment")
		}
	}

	searchDirs = append(searchDirs, "/usr/local/share/hub", "/usr/share/hub")

	for i := len(what); i > 0; i-- {
		script := "hub-" + strings.Join(what[0:i], "-")
		newArgs := append(what[i:], args...)

		if config.Trace {
			log.Printf("Trying %s with args %v", script, newArgs)
		}

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
				return path, newArgs, nil
			}
		}

		path, err := exec.LookPath(script)
		if err == nil {
			return path, newArgs, nil
		}
	}

	printCustomHubDir := ""
	if customHubDir != "" {
		printCustomHubDir = fmt.Sprintf(", $HUB_EXTENSIONS=%s", customHubDir)
	}

	maybeInstall := ""
	if customHubDir == "" && homeHubDir != "" {
		_, err := os.Stat(homeHubDir)
		verb := "update"
		if err != nil {
			if util.NoSuchFile(err) {
				verb = "install"
			}
		}
		maybeInstall = fmt.Sprintf("\n\t%s Hub CTL extensions with `hub extensions %[1]s`?", verb)
	}

	return "", nil, fmt.Errorf("Extension not found in %v%s, nor $PATH%s", searchDirs, printCustomHubDir, maybeInstall)
}

func RunExtension(what, args []string) {
	code, err := runExtension(what, args)
	if err != nil {
		log.Fatalf("Unable to call %v extension: %v", what, err)
	}
	os.Exit(code)
}

func runExtension(what, args []string) (int, error) {
	if config.Debug {
		log.Printf("Searching extension %v with args %v", what, args)
	}
	executable, args, err := ExtensionPath(what, args)
	if err != nil {
		return 0, err
	}
	if config.Debug {
		log.Printf("Found extension %s %v", executable, args)
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
