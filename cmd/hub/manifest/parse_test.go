package manifest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const manifestFilepath string = "../../../test/cmd/hub/manifest/"

func TestManifestParse(t *testing.T) {
	manifestFile := filepath.Join(manifestFilepath, "single-manifest.yaml")
	manifest, manifests, filename, err := ParseManifest([]string{manifestFile})

	assert.Nil(t, err, "ManifestParse should not return error")
	assert.Equal(t, manifestFile, filename, "ManifestParse should return filename equal to %s", manifestFile)
	assert.NotNil(t, manifest, "ManifestParse returned manifest should not be nil")
	assert.Nil(t, manifests, "ManifestParse returned manifests should be nil")

	manifestFile = filepath.Join(manifestFilepath, "multiple-yamls-in-manifest.yaml")
	manifest, manifests, filename, err = ParseManifest([]string{manifestFile})
	assert.Nil(t, err, "ManifestParse should not return error")
	assert.Equal(t, manifestFile, filename, "ManifestParse should return filename equal to %s", manifestFile)
	assert.NotNil(t, manifest, "ManifestParse returned manifest should not be nil")
	assert.NotNil(t, manifests, "ManifestParse returned manifests should not be nil")
}

func TestGenerateLifecycleOrder(t *testing.T) {
	manifestsFile, manifestsFiles, _, _ := ParseManifest([]string{filepath.Join(manifestFilepath, "manifest_with_errors.yaml")})
	manifestsFiles = append(manifestsFiles, *manifestsFile)
	for _, file := range manifestsFiles {
		order, err := GenerateLifecycleOrder(&file)
		t.Log(err)
		assert.NotNil(t, err, "GenerateLifecycleOrder should return error", err)
		assert.Nil(t, order, "Generated lifecycle order should be nil")
	}

	manifestWithLifecycleOrder, _, _, err := ParseManifest([]string{filepath.Join(manifestFilepath, "manifest_with_lifecycle_order.yaml")})
	assert.Nil(t, err, "ManifestParse should not return error")
	manifestWithoutLifecycleOrder, _, _, err := ParseManifest([]string{filepath.Join(manifestFilepath, "manifest_without_lifecycle_order.yaml")})
	assert.Nil(t, err, "ManifestParse should not return error")

	order, err := GenerateLifecycleOrder(manifestWithoutLifecycleOrder)
	assert.Nil(t, err, "GenerateLifecycleOrder should not return error")
	assert.ElementsMatch(t, manifestWithLifecycleOrder.Lifecycle.Order, order, "Generated lifecycle order should match manifest lifecycle order ")

	exactOrder, err := GenerateLifecycleOrder(manifestWithLifecycleOrder)
	assert.Nil(t, err, "GenerateLifecycleOrder should not return error")
	assert.Exactly(t, manifestWithLifecycleOrder.Lifecycle.Order, exactOrder, "Generated lifecycle order should match manifest lifecycle order")
}
