// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const remoteUrl = "https://github.com/epam/hubctl.git"
const branchRef = "master"
const tagRef = "v1.0.0"

func TestClone(t *testing.T) {
	dir := t.TempDir()

	err := Clone(remoteUrl, branchRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")
}

func TestPullBranch(t *testing.T) {
	dir := t.TempDir()

	err := Clone(remoteUrl, branchRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")

	err = Pull("", dir)
	assert.NoError(t, err, "should return nil error if repo and using HEAD ref")

	err = Pull(branchRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")
}

func TestPullTag(t *testing.T) {
	dir := t.TempDir()

	err := Clone(remoteUrl, tagRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")

	err = Pull("", dir)
	assert.NoError(t, err, "should return nil error if repo exist and using HEAD ref")

	err = Pull(tagRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")
}

func TestFindRemoteBranch(t *testing.T) {
	ref, err := findRemoteReference(remoteUrl, "develop")

	assert.True(t, ref.Name().IsBranch(), "should return reference as branch")
	assert.Equal(t, "refs/heads/develop", ref.Name().String(), "ref full name should equals to refs/heads/develop")
	assert.NoError(t, err, "should return nil error if repo branch/tag is exist")

	ref, err = findRemoteReference(remoteUrl, "v1.0.0")

	assert.True(t, ref.Name().IsTag(), "should return reference as tag")
	assert.Equal(t, "refs/tags/v1.0.0", ref.Name().String(), "ref full name should equals to refs/tags/v1.0.0")
	assert.NoError(t, err, "should return nil error if repo branch/tag is exist")

	ref, err = findRemoteReference(remoteUrl, "i-hope-there-is-no-such-branch-or-tag")

	assert.Nil(t, ref, "ref should be nil")
	assert.Error(t, err, "should return error if repo branch/tag does not exist")
}
