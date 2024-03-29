// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ext

import (
	"testing"

	goGit "github.com/go-git/go-git/v5"
	goGitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

func TestExtensionsGitRepoUrlIsValid(t *testing.T) {
	rem := goGit.NewRemote(memory.NewStorage(), &goGitConfig.RemoteConfig{
		Name: "origin",
		URLs: []string{ExtensionsGitRemote},
	})

	_, err := rem.List(&goGit.ListOptions{})

	assert.NoError(t, err, "When extensions git repository URL is valid, git ls-remote should not return error")
}

func TestExtensionsInstallAndUpdateBranchChannel(t *testing.T) {
	dir := t.TempDir()

	t.Log("Install extensions")
	err := Install(ExtensionsGitRemote, ExtensionsRef, dir)
	assert.NoError(t, err, "When install extensions it should not panic")

	t.Log("Update extensions")
	err = Update(dir)
	assert.NoError(t, err, "When update extensions, it should not panic")
}

func TestExtensionsInstallAndUpdateTagChannel(t *testing.T) {
	dir := t.TempDir()

	t.Log("Install extensions")
	err := Install(ExtensionsGitRemote, "stable", dir)
	assert.NoError(t, err, "When install extensions it should not panic")

	t.Log("Update extensions")
	err = Update(dir)
	assert.NoError(t, err, "When update extensions, it should not panic")
}
