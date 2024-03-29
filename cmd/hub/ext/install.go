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

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/git"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	ExtensionsGitRemote = "https://github.com/epam/hub-extensions.git"
	ExtensionsRef       = "master"
)

func defaultExtensionsDir() string {
	return filepath.Join(os.Getenv("HOME"), hubDir)
}

func Install(repo, ref, dir string) error {
	if dir == "" {
		dir = defaultExtensionsDir()
	}

	_, err := os.Stat(filepath.Join(dir, ".git"))
	if err == nil {
		util.Warn("`%s` already exist; try `hubctl extensions update`?", dir)
		return nil
	}

	if config.Debug {
		log.Printf("Cloning extensions repository: %s", repo)
	}

	err = git.Clone(repo, ref, dir)

	if err != nil {
		return fmt.Errorf("unable to install extensions to `%s` directory: %v", dir, err)
	}

	postInstall(dir)

	if config.Verbose {
		log.Printf("Hub CTL extensions installed into %s", dir)
	}

	return nil
}

func Update(dir string) error {
	if dir == "" {
		dir = defaultExtensionsDir()
	}

	err := git.Pull("", dir)
	if err != nil {
		return fmt.Errorf("unable to update extensions in `%s` directory: %v", dir, err)
	}

	postInstall(dir)

	if config.Verbose {
		log.Printf("Hub CTL extensions updated in %s", dir)
	}

	return nil
}

func postInstall(dir string) {
	hook := filepath.Join(dir, "post-install")
	_, err := os.Stat(hook)
	if err == nil {
		cmd := exec.Cmd{
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
}
