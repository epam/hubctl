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

const (
	testRemote = "https://github.com/epam/hubctl.git"
	testRef    = "master"
)

func TestStatusOfInvalidGitRepo(t *testing.T) {
	dir := t.TempDir()
	clean, err := Status(dir)

	assert.False(t, clean, "should return clean as false if directory is empty")
	assert.Error(t, err, "should return error if directory is empty")

	err = Clone(testRemote, testRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")

	clean, err = Status(dir)

	assert.True(t, clean, "should return clean as true if directory is valid git repo")
	assert.NoError(t, err, "should return nil error if directory is valid git repo")
}

func TestHeadOf(t *testing.T) {
	dir := t.TempDir()

	name, rev, err := HeadInfo(dir)

	assert.Empty(t, name, "should return empty name if directory is empty")
	assert.Empty(t, rev, "should return empty revision if directory is empty")
	assert.Error(t, err, "should return an error if directory is empty")

	err = Clone(testRemote, testRef, dir)
	assert.NoError(t, err, "should return nil error if repo and ref exist")

	name, rev, err = HeadInfo(dir)

	assert.NotEmpty(t, name, "should not return empty name if directory is valid git repo")
	assert.NotEmpty(t, rev, "should not be empty rev if directory is valid git repo")
	assert.NoError(t, err, "should not return an error if directory is valid git repo")
}
