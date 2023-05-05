// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"fmt"
	"log"

	"github.com/epam/hubctl/cmd/hub/util"
	goGit "github.com/go-git/go-git/v5"
	goGitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

func Clone(remote, ref, dir string) error {
	referenceName, err := findRemoteBranch(remote, ref)

	if err != nil {
		return fmt.Errorf("unable to clone git repo %s at `%s` into `%s`: %v", remote, ref, dir, err)
	}

	_, err = goGit.PlainClone(dir, false, &goGit.CloneOptions{
		URL:               remote,
		ReferenceName:     referenceName,
		SingleBranch:      true,
		Progress:          log.Default().Writer(),
		Tags:              goGit.NoTags,
		RecurseSubmodules: goGit.NoRecurseSubmodules,
	})

	if err != nil {
		return fmt.Errorf("unable to clone git repo %s at `%s` into `%s`: %v", remote, ref, dir, err)
	}

	return nil
}

const remoteName = "origin"

func pullErrMsgFormat(dir string, err error) error {
	return fmt.Errorf("unable to pull git repo in `%s` directory: %v", dir, err)
}

func Pull(ref, dir string) error {
	repo, err := goGit.PlainOpen(dir)
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	repoConfig, err := repo.Config()
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	remote := repoConfig.Remotes[remoteName].URLs[0]

	referenceName, err := findRemoteBranch(remote, ref)
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	err = worktree.Pull(&goGit.PullOptions{
		RemoteName:        remoteName,
		ReferenceName:     referenceName,
		SingleBranch:      true,
		Progress:          log.Default().Writer(),
		RecurseSubmodules: goGit.NoRecurseSubmodules,
	})

	if err != nil {
		if err == goGit.NoErrAlreadyUpToDate {
			log.Printf("%v", err)
		} else {
			return pullErrMsgFormat(dir, err)
		}
	}

	return nil
}

const (
	refPrefix     = "refs/"
	refHeadPrefix = refPrefix + "heads/"
	refTagPrefix  = refPrefix + "tags/"
)

var refPrefixes = []string{refHeadPrefix, refTagPrefix}

var refFindOrder = []string{
	refHeadPrefix + "%s",
	refTagPrefix + "%s",
}

func findRemoteBranch(remote, targetRef string) (plumbing.ReferenceName, error) {
	rem := goGit.NewRemote(memory.NewStorage(), &goGitConfig.RemoteConfig{
		Name: remoteName,
		URLs: []string{remote},
	})

	refs, err := rem.List(&goGit.ListOptions{})

	if err != nil {
		return "", fmt.Errorf("unable to get ref list of `%s` remote: %v", remote, err)
	}

	for _, refNameFormat := range refFindOrder {
		for _, ref := range refs {
			name := ref.Name().String()
			if util.ContainsPrefix(refPrefixes, name) {
				if (util.ContainsPrefix(refPrefixes, targetRef) && targetRef == name) || fmt.Sprintf(refNameFormat, targetRef) == name {
					return ref.Name(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("unable to find git ref `%s` in `%s` remote", targetRef, remote)
}
