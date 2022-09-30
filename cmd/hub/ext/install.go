// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ext

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/git"
	"github.com/epam/hubctl/cmd/hub/util"
)

const extensionsGitRemote = "https://github.com/epam/hubctl-extensions.git"

func defaultExtensionsDir() string {
	return filepath.Join(os.Getenv("HOME"), hubDir)
}

func Install(dir string) {
	if dir == "" {
		dir = defaultExtensionsDir()
	}

	_, err := os.Stat(filepath.Join(dir, ".git"))
	if err == nil {
		util.Warn("`%s` already exist; try `hub extensions update`?", dir)
	}

	cmd := exec.Cmd{
		Path:   git.GitBinPath(),
		Args:   []string{"git", "clone", "-b", "stable", extensionsGitRemote, dir},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if config.Debug {
		log.Printf("Cloning extensions repository: %v", cmd.Args)
	}
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Unable to git clone %s into %s: %v", extensionsGitRemote, dir, err)
	}

	hook := filepath.Join(dir, "post-install")
	_, err = os.Stat(hook)
	if err == nil {
		cmd = exec.Cmd{
			Path:   "post-install",
			Dir:    dir,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err = cmd.Run()
		if err != nil {
			util.Warn("Unable to run post-install hook in %s: %v", dir, err)
		}
	} else {
		util.Warn("No post-install hook %s: %v", hook, err)
	}

	if config.Verbose {
		log.Printf("Hub CTL extensions installed into %s", dir)
	}
}

func Update(dir string) {
	if dir == "" {
		dir = defaultExtensionsDir()
	}

	hook := filepath.Join(dir, "update")
	_, err := os.Stat(hook)
	if err == nil {
		cmd := exec.Cmd{
			Path:   "update",
			Dir:    dir,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err = cmd.Run()
		if err != nil {
			util.Warn("Unable to run update in %s: %v", dir, err)
		}
	} else {
		log.Fatalf("No update hook %s: %v", hook, err)
	}

	if config.Verbose && err == nil {
		log.Printf("Hub CTL extensions updated in %s", dir)
	}
}
