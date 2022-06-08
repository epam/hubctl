package lifecycle

import (
	"testing"

	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/stretchr/testify/assert"
)

func TestMaybeHooks(t *testing.T) {
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
	res := findRelevantHooks("pre-deploy", hooks)
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
	res = findRelevantHooks("pre-backup", hooks)
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
	res = findRelevantHooks("post-deploy", hooks)
	assert.Equal(t, len(res), 1)
	assert.Equal(t, res[0].File, "/etc/hook3")
}
