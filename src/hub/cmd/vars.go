package cmd

const (
	envVarNameHubCli            = "HUB"
	envVarNameElaborate         = "HUB_ELABORATE"
	envVarNameState             = "HUB_STATE"
	envVarNameAwsRegion         = "HUB_AWS_REGION"
	envVarNameComponentsBaseDir = "HUB_COMPONENTS_BASEDIR"
	envVarNameHubApi            = "HUB_API"
	SuperHubIo                  = ".superhub.io"
)

var supportedCloudAccountKinds = []string{"aws", "azure", "gcp"}

var (
	name                  string
	environmentSelector   string
	templateSelector      string
	componentName         string
	componentsBaseDir     string
	elaborateManifest     string
	stateManifest         string
	stateManifestExplicit string
	environmentOverrides  string
	dryRun                bool
	osEnvironmentMode     string
	pipeOutputInRealtime  bool
	outputFiles           string
	waitAndTailDeployLogs bool
	showSecrets           bool
	jsonFormat            bool
	patchReplace          bool
	patchRaw              bool
)
