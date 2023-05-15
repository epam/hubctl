// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"fmt"
	"log"
	"strings"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func HeadInfo(dir string) (string, string, error) {
	what := "HEAD"
	name := "(unknown)"
	rev := "(unknown)"

	repo, err := goGit.PlainOpen(dir)
	if err != nil {
		return "", "", fmt.Errorf("directory %s is not valid git repository: %v", dir, err)
	}

	ref, err := repo.Reference(plumbing.ReferenceName(what), true)
	if err != nil {
		return "", "", fmt.Errorf("unable to get git repository %s %s name: %v", dir, what, err)
	}

	name = strings.Trim(ref.Name().String(), "\r\n")
	rev = strings.Trim(ref.Hash().String(), "\r\n")

	return name, rev, nil
}

func Status(dir string) (bool, error) {
	repo, err := goGit.PlainOpen(dir)
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
