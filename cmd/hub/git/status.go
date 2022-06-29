// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

func Status(dir string) (bool, string, error) {
	clean := false
	status := "(unknown)"
	var out bytes.Buffer
	cmd := exec.Cmd{
		Path:   GitBinPath(),
		Dir:    dir,
		Args:   []string{"git", "status"},
		Stdout: &out,
	}
	gitDebug2(&cmd, &out)
	err := cmd.Run()
	if err != nil {
		return clean, status, fmt.Errorf("Unable to determine Git repo `%s` status: %v", dir, err)
	}
	status = out.String()
	clean = strings.Contains(status, "nothing to commit, working tree clean")
	return clean, status, nil
}
