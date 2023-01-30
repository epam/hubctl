package lifecycle

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/stretchr/testify/assert"
)

// Create a temp directory and seed an extension
func setup(t *testing.T) {
	extensions := t.TempDir()
	extension := path.Join(extensions, "hub-component-arm")
	_, err := os.Create(extension)
	if err != nil {
		log.Fatal(err)
	}
	os.Setenv("HUB_EXTENSIONS", extensions)
}

func TestProbeARM(t *testing.T) {
	setup(t)
	requiresFoo := manifest.Manifest{
		Requires: []string{"foo"},
	}

	found, err := probeImplementation("./foo", "deploy", &requiresFoo)
	if err != nil {
		assert.Errorf(t, err, "When 'foo' in requires is not a well known, should not find implementation and fail with error", err)
	}
	assert.Falsef(t, found, "When 'foo' in requires is not a well known, should not find implementation and fail with error")

	requiresARM := manifest.Manifest{
		Requires: []string{"arm"},
	}
	found, err = probeImplementation("./foo", "deploy", &requiresARM)
	if err != nil {
		assert.NoErrorf(t, err, "When 'arm' in component requires, it should resolve to implementation %v", err)
	}
	assert.Truef(t, found, "When 'arm' in component requires, it should resolve to implementation")
}

// ARM extension should be called when component requires 'arm' in the manifest
func TestFindARMImplementation(t *testing.T) {
	setup(t)
	requiresARM := manifest.Manifest{
		Requires: []string{"arm"},
	}
	extension, err := findImplementation("./foo", "deploy", &requiresARM)
	if err != nil {
		assert.Fail(t, "When 'arm' in component requires, it should resolve to extension. Error is not expected here. Error is %v", err)
	}
	assert.NotNil(t, extension, "When 'arm' in component requires, it should resolve to the extension")

	assert.Contains(t, extension.Path, "/hub-component-arm", "ARM extension should be returned")
	assert.FileExists(t, extension.Path, "ARM should extension should be found in the file system")
}
