package lifecycle

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

var helmVerTpl = "version.BuildInfo{Version:\"%s\", GitCommit:\"50f003e5ee8704ec937a756c646870227d7c8b58\", GitTreeState:\"clean\", GoVersion:\"go1.18.8\"}"

func formatHelmVersion(version string) []byte {
	return []byte(fmt.Sprintf(helmVerTpl, version))
}

func TestCheckRequiresBinVersion(t *testing.T) {
	min, _ := version.NewVersion("3.5.2")
	helm := binVersion["helm"]
	helm.minVersion = min

	raw_data := formatHelmVersion(min.String())
	err := checkRequiresBinVersion(helm, raw_data)
	assert.Error(t, err, "When versions are equal, checkRequiresBinVersion should not return validation error")

	raw_data = formatHelmVersion("v0.0.1")
	err = checkRequiresBinVersion(helm, raw_data)
	assert.Error(t, err, "When version is less than required, checkRequiresBinVersion should return validation error")

	raw_data = formatHelmVersion("v100.0.0")
	err = checkRequiresBinVersion(helm, raw_data)
	assert.NoError(t, err, "When version is greater than required, checkRequiresBinVersion should not return validation error")

	raw_data = formatHelmVersion("v3.10.2")
	err = checkRequiresBinVersion(helm, raw_data)
	assert.NoError(t, err, "When version number starts with 1 but actually is 10 there should be no error")

}
