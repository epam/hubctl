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
