package ext

import (
	"os"
	"os/exec"
	"testing"

	"github.com/epam/hubctl/cmd/hub/git"
	"github.com/stretchr/testify/assert"
)

func TestExtensionsGitRepoUrlIsValid(t *testing.T) {
	cmd := exec.Cmd{
		Path:   git.GitBinPath(),
		Args:   []string{"git", "ls-remote", "--tags", "--quiet", extensionsGitRemote},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	err := cmd.Run()
	assert.NoError(t, err, "When extensions git repository URL is valid, git ls-remote should not return error")
}
