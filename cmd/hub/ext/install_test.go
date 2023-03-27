package ext

import (
	"log"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

func TestExtensionsGitRepoUrlIsValid(t *testing.T) {
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{extensionsGitRemote},
	})

	_, err := rem.List(&git.ListOptions{})

	assert.NoError(t, err, "When extensions git repository URL is valid, git ls-remote should not return error")
}

func TestExtensionsInstall(t *testing.T) {
	assert.NotPanics(t, func() {
		Install(t.TempDir())
	}, "When install extensions, it should not panic")
}

func TestExtensionsInstallAndUpdate(t *testing.T) {
	assert.NotPanics(t, func() {
		dir := t.TempDir()
		log.Print("Install extensions")
		Install(dir)
		log.Print("Update extensions")
		Update(dir)
	}, "When install extensions and then update them, it should not panic")
}
