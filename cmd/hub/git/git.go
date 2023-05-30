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

func cloneErrMsgFormat(remoteUrl, ref, dir string, err error) error {
	return fmt.Errorf("unable to clone git repo %s at `%s` into `%s`: %v", remoteUrl, ref, dir, err)
}

func Clone(remoteUrl, ref, dir string) error {
	reference, err := findRemoteReference(remoteUrl, ref)
	if err != nil {
		return cloneErrMsgFormat(remoteUrl, ref, dir, err)
	}

	_, err = goGit.PlainClone(dir, false, &goGit.CloneOptions{
		URL:               remoteUrl,
		ReferenceName:     reference.Name(),
		SingleBranch:      true,
		Progress:          log.Default().Writer(),
		Tags:              goGit.NoTags,
		RecurseSubmodules: goGit.NoRecurseSubmodules,
	})

	if err != nil {
		return cloneErrMsgFormat(remoteUrl, ref, dir, err)
	}

	return nil
}

const remoteName = "origin"

func pullErrMsgFormat(dir string, err error) error {
	return fmt.Errorf("unable to pull git repo in `%s` directory: %v", dir, err)
}

func Pull(targetRef, dir string) error {
	repo, err := goGit.PlainOpen(dir)
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	reference, err := repo.Head()
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	if targetRef == "" {
		refs, err := repo.References()
		if err != nil {
			return pullErrMsgFormat(dir, err)
		}
		defer refs.Close()

		for ref, err := refs.Next(); err == nil; {
			if ref.Hash() == reference.Hash() {
				reference = ref
				break
			}
		}
	} else {
		remote, err := repo.Remote(remoteName)
		if err != nil {
			return pullErrMsgFormat(dir, err)
		}

		remoteUrl := remote.Config().URLs[0]
		reference, err = findRemoteReference(remoteUrl, targetRef)
		if err != nil {
			return pullErrMsgFormat(dir, err)
		}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return pullErrMsgFormat(dir, err)
	}

	err = worktree.Pull(&goGit.PullOptions{
		RemoteName:        remoteName,
		ReferenceName:     reference.Name(),
		SingleBranch:      true,
		Progress:          log.Default().Writer(),
		RecurseSubmodules: goGit.NoRecurseSubmodules,
	})

	if err != nil {
		if err == goGit.NoErrAlreadyUpToDate {
			log.Print(err)
			return nil
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

func findRemoteReference(remoteUrl, targetRef string) (*plumbing.Reference, error) {
	remote := goGit.NewRemote(memory.NewStorage(), &goGitConfig.RemoteConfig{
		Name: remoteName,
		URLs: []string{remoteUrl},
	})

	refs, err := remote.List(&goGit.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("unable to get ref list of `%s` remote: %v", remoteUrl, err)
	}

	for _, refNameFormat := range refFindOrder {
		for _, ref := range refs {
			name := ref.Name().String()
			if util.ContainsPrefix(refPrefixes, name) {
				targetName := fmt.Sprintf(refNameFormat, targetRef)
				if (util.ContainsPrefix(refPrefixes, targetRef) && targetRef == name) || targetName == name {
					return ref, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unable to find git ref `%s` in `%s` remote", targetRef, remoteUrl)
}
