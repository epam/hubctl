package cmd

const (
	envVarNameHubCli            = "HUB"
	envVarNameElaborate         = "HUB_ELABORATE"
	envVarNameState             = "HUB_STATE"
	envVarNameAwsRegion         = "HUB_AWS_REGION"
	envVarNameComponentsBaseDir = "HUB_COMPONENTS_BASEDIR"
	envVarNameHubApi            = "HUB_API"
)

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
	testVerb              bool
	osEnvironmentMode     string
	pipeOutputInRealtime  bool
	outputFiles           string
	waitAndTailDeployLogs bool
)
