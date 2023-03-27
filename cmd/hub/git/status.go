// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
)

func HeadInfo(dir string) (string, string, error) {
	what := "HEAD"
	name := "(unknown)"
	rev := "(unknown)"
	var out bytes.Buffer
	gitBin := GitBinPath()

	cmd := exec.Cmd{
		Path:   gitBin,
		Dir:    dir,
		Args:   []string{"git", "name-rev", "--name-only", what},
		Stdout: &out,
	}
	gitDebug2(&cmd, &out)
	err := cmd.Run()
	if err != nil {
		return name, rev,
			fmt.Errorf("Unable to determine Git repo `%s` HEAD name: %v", dir, err)
	}
	name = strings.Trim(out.String(), "\r\n")

	out.Truncate(0)
	cmd = exec.Cmd{
		Path:   gitBin,
		Dir:    dir,
		Args:   []string{"git", "rev-parse", what},
		Stdout: &out,
	}
	gitDebug2(&cmd, &out)
	err = cmd.Run()
	if err != nil {
		return name, rev,
			fmt.Errorf("Unable to determine Git repo `%s` HEAD hash: %v", dir, err)
	}
	rev = strings.Trim(out.String(), "\r\n")

	return name, rev, nil
}

func Status(dir string) (bool, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return false, fmt.Errorf("directory %s is not valid git repository: %v", dir, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("git repository %s has invalid worktree: %v", dir, err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("unable to determine git repo %s status: %v", dir, err)
	}

	if status.String() != "" {
		log.Printf("Git repository status: %s", status)
	}

	return status.IsClean(), nil
}
