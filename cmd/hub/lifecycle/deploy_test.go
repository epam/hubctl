// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"os"
	"testing"

	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/stretchr/testify/assert"
)

func TestHooksFilterByTrigger(t *testing.T) {
	hooks := []manifest.Hook{
		{
			File: "/etc/hook1",
			Triggers: []string{
				"pre-*",
			},
		},
		{
			File: "/etc/hook2",
			Triggers: []string{
				"pre-undeploy",
				"*-deploy",
			},
		},
		{
			File: "/etc/hook3",
			Triggers: []string{
				"post-deploy",
			},
		},
	}
	res := findHooksByTrigger("pre-deploy", hooks)
	assert.Equal(t, len(res), 2)
	hooks = []manifest.Hook{
		{
			File: "/etc/hook1",
			Triggers: []string{
				"*",
			},
		},
		{
			File: "/etc/hook2",
			Triggers: []string{
				"pre-undeploy",
				"*-deploy",
			},
		},
		{
			File: "/etc/hook3",
			Triggers: []string{
				"post-deploy",
			},
		},
	}
	res = findHooksByTrigger("pre-backup", hooks)
	assert.Equal(t, len(res), 1)
	assert.Equal(t, res[0].File, "/etc/hook1")
	hooks = []manifest.Hook{
		{
			File: "/etc/hook1",
			Triggers: []string{
				"pre-*",
			},
		},
		{
			File: "/etc/hook2",
			Triggers: []string{
				"pre-undeploy",
				"*-backup",
			},
		},
		{
			File: "/etc/hook3",
			Triggers: []string{
				"post-deploy",
			},
		},
	}
	res = findHooksByTrigger("post-deploy", hooks)
	assert.Equal(t, len(res), 1)
	assert.Equal(t, res[0].File, "/etc/hook3")
}

func TestFindScript(t *testing.T) {
	temp := t.TempDir()
	// always := []string{"*"}
	result, err := findScript(temp+"/foo", temp)
	assert.Empty(t, result, "It should not return value when script is not found")
	assert.Error(t, err, "It should return error when script is not found")

	helloWorld := []byte(`#!/bin/sh -e
	echo "Hello, World"`)
	os.WriteFile(temp+"/bar", helloWorld, 0755)

	result, err = findScript(temp+"/bar", temp)
	assert.NotEmpty(t, result, "It should return path to the actual file")
	assert.FileExists(t, result, "It should return path to the actual file")
	assert.NoError(t, err)

	result, err = findScript("bar", temp)
	assert.FileExists(t, result, "It should return path to the actual file")
	assert.NoError(t, err)

	os.WriteFile(temp+"/baz.sh", helloWorld, 0755)
	result, err = findScript("baz", temp)
	assert.FileExists(t, result, "It should return path to the actual file when extension is omitted")
	assert.NoError(t, err)
}
