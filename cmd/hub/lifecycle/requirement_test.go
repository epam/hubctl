package lifecycle

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var outputTemplates = map[string]string{
	"aws": "aws-cli/%s Python/3.11.5 Linux/5.15.90.1 source/x86_64.ubuntu.22 prompt/off",
	"az": `{
		"azure-cli": "%s",
		"azure-cli-core": "2.52.0",
		"azure-cli-telemetry": "1.1.0",
		"extensions": {}
	}`,
	"gcloud": `Google Cloud SDK %s
alpha 2023.09.13
beta 2023.09.13
bq 2.0.98
bundled-python3-unix 3.9.16
core 2023.09.13
gcloud-crc32c 1.0.0
gsutil 5.25`,
	"gsutil": "version: %s",
	"vault":  "Vault v%s ('56debfa71653e72433345f23cd26276bc90629ce+CHANGES'), built 2023-09-11T21:23:55Z",
	"kubectl": `{
		"clientVersion": {
		  "major": "1",
		  "minor": "28",
		  "gitVersion": "v%s",
		  "gitCommit": "8dc49c4b984b897d423aab4971090e1879eb4f23",
		  "gitTreeState": "clean",
		  "buildDate": "2023-08-24T11:16:29Z",
		  "goVersion": "go1.20.7",
		  "compiler": "gc",
		  "platform": "linux/amd64"
		},
		"kustomizeVersion": "v5.0.4-0.20230601165947-6ce0bf390ce3"
	  }`,
	"helm":      "version.BuildInfo{Version:\"v%s\", GitCommit:\"3a31588ad33fe3b89af5a2a54ee1d25bfe6eaa5e\", GitTreeState:\"clean\", GoVersion:\"go1.20.7\"}",
	"terraform": "Terraform v%s\non linux_amd64",
}

func formatVersion(binary, version string) []byte {
	return []byte(fmt.Sprintf(outputTemplates[binary], version))
}

func testRequiredbinaryVersion(binary, equalVersion, newerVersion, olderVersion, startFromTen string, t *testing.T) {
	reqVer := binVersion[binary]
	err := checkRequiresBinVersion(reqVer, formatVersion(binary, equalVersion))
	assert.NoError(t, err, "When version is equal with minimal required, checkRequiresBinVersion should not return validation error")
	err = checkRequiresBinVersion(reqVer, formatVersion(binary, newerVersion))
	assert.NoError(t, err, "When version is greater than minimal required, checkRequiresBinVersion should not return validation error")
	err = checkRequiresBinVersion(reqVer, formatVersion(binary, olderVersion))
	assert.Error(t, err, "When version is less than minimal required, checkRequiresBinVersion should return validation error")
	err = checkRequiresBinVersion(reqVer, formatVersion(binary, startFromTen))
	assert.NoError(t, err, "When version number starts with 1 but actually is 10, checkRequiresBinVersion should not return validation error")
}

func TestCheckAwsVersion(t *testing.T) {
	testRequiredbinaryVersion("aws", "2.10", "2.13.19", "2.0.2", "10.1.1", t)
}

func TestCheckAzureVersion(t *testing.T) {
	testRequiredbinaryVersion("az", "2.40", "2.52.0", "2.0.2", "10.1.1", t)
}

func TestCheckGcloudVersion(t *testing.T) {
	testRequiredbinaryVersion("gcloud", "400.0.0", "446.0.1", "140.0.2", "1010.1.1", t)
}

func TestCheckGsutilVersion(t *testing.T) {
	testRequiredbinaryVersion("gsutil", "5.0", "5.25", "3.52", "10.1.1", t)
}

func TestCheckVaultVersion(t *testing.T) {
	testRequiredbinaryVersion("vault", "1.10.0", "1.14.3", "1.9.10", "10.1.1", t)
}

func TestCheckKubectlVersion(t *testing.T) {
	testRequiredbinaryVersion("kubectl", "1.19", "1.28.1", "1.18.15", "10.1.1", t)
}

func TestCheckHelmVersion(t *testing.T) {
	testRequiredbinaryVersion("helm", "3.11", "3.12.3", "3.5.1", "10.1.1", t)
}

func TestCheckTerraformVersion(t *testing.T) {
	testRequiredbinaryVersion("terraform", "1.0", "1.5.7", "0.14.1", "10.1.1", t)
}
