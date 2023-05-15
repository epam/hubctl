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
const ref = "master"

func TestClone(t *testing.T) {
	dir := t.TempDir()

	err := Clone(remoteUrl, ref, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")
}

func TestPull(t *testing.T) {
	dir := t.TempDir()

	err := Clone(remoteUrl, ref, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")

	err = Pull(ref, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")
}

func TestFindRemoteBranch(t *testing.T) {
	refName, err := findRemoteBranch(remoteUrl, "develop")

	assert.True(t, refName.IsBranch(), "should return reference as branch")
	assert.Equal(t, "refs/heads/develop", refName.String(), "ref full name should equals to refs/heads/develop")
	assert.NoError(t, err, "should return nil error if repo branch/tag is exist")

	refName, err = findRemoteBranch(remoteUrl, "v1.0.0")

	assert.True(t, refName.IsTag(), "should return reference as tag")
	assert.Equal(t, "refs/tags/v1.0.0", refName.String(), "ref full name should equals to refs/tags/v1.0.0")
	assert.NoError(t, err, "should return nil error if repo branch/tag is exist")

	refName, err = findRemoteBranch(remoteUrl, "i-hope-there-is-no-such-branch-or-tag")

	assert.Empty(t, refName, "ref should be empty")
	assert.Error(t, err, "should return error if repo branch/tag does not exist")
}
