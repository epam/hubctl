package git

import (
	"testing"

	"github.com/epam/hubctl/cmd/hub/ext"
	"github.com/stretchr/testify/assert"
)

func TestStatusOfInvalidGitRepo(t *testing.T) {
	dir := t.TempDir()
	clean, err := Status(dir)

	assert.False(t, clean, "should return clean as false if directory is empty")
	assert.Error(t, err, "should return error if directory is empty")

	ext.Install(dir)

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

	ext.Install(dir)

	name, rev, err = HeadInfo(dir)

	assert.NotEmpty(t, name, "Git reference name should not be empty in valid git repo")
	assert.NotEmpty(t, rev, "Git reference revision should not be empty in valid git repo")
	assert.NoError(t, err, "Git HeadOf of valid git repo should not produce an error")
}
