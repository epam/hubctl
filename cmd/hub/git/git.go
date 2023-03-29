// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
)

var dirMode = os.FileMode(0755)

func GitBinPath() string {
	return gitSubCommandBinPath("git")
}

func gitSubCommandBinPath(command string) string {
	path, err := exec.LookPath(command)
	if err != nil {
		if config.Trace {
			log.Printf("Git binary `%s` lookup: %v; using %s", command, err, config.GitBinDefault)
		}
		path = config.GitBinDefault
	}
	return path
}

func printGitArgs(cmd *exec.Cmd) {
	log.Printf("%s (%s)", strings.Join(cmd.Args, " "), cmd.Dir)
}

func gitDebug(cmd *exec.Cmd) {
	if config.Debug {
		cmd.Stderr = os.Stdout
		if config.Trace {
			cmd.Stdout = os.Stdout
			printGitArgs(cmd)
		}
	}
}
