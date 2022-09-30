// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/epam/hubctl/cmd/hub/git"
	"github.com/epam/hubctl/cmd/hub/util"
)

func gitStatus(dir string, calculateStatus bool) (map[string]string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("Unable to calculate absolute path of `%s`: %v", dir, err)
	}
	dir = abs
	for {
		gitDir := filepath.Join(dir, ".git")
		_, err = os.Stat(gitDir)
		if err != nil {
			if util.NoSuchFile(err) {
				parent := filepath.Dir(dir)
				if dir == parent {
					return map[string]string{
						"ref":   "(not a Git)",
						"clean": "",
					}, nil
				}
				dir = parent
				continue
			}
			return nil, fmt.Errorf("Unable to stat `%s`: %v", gitDir, err)
		}
		break
	}
	name, rev, err := git.HeadInfo(dir)
	if err != nil {
		return nil, err
	}
	clean := "not calculated"
	if calculateStatus {
		isClean, _, err := git.Status(dir)
		if err != nil {
			util.Warn("%v", err)
		} else {
			if isClean {
				clean = "clean"
			} else {
				clean = "dirty"
			}
		}
	}
	if len(rev) == 40 {
		rev = rev[:7]
	}
	return map[string]string{
		"ref":   fmt.Sprintf("%s %s", name, rev),
		"clean": clean,
	}, nil
}
