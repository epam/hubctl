package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "hub",
	Short: "Hub CLI is a stack composition and lifecycle tool",
	Long: `Hub CLI is a stack composition and lifecycle tool:
- template and stack creation, stack deploy / undeploy / backup lifecycle;
- stack and component parameters, output variables, and status;
- management of templates, stacks, components on SuperHub.io`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Update()
		if config.Debug {
			log.Printf("Hub CLI %s %s\n", util.CliVersion, runtime.Version())
		}
		maybeMeterCommand(cmd)
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		util.PrintAllWarnings()
	},
}

func Execute() {
	ctx := context.WithValue(context.Background(), contextKey, &CmdContext{})
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "Config file (default is $HOME/.hub-config.{yaml,json})")
	RootCmd.PersistentFlags().StringVar(&config.CacheFile, "cache", "", "API cache file (default is $HOME/.hub-cache.yaml)")

	apiDefault := os.Getenv(envVarNameHubApi)
	if apiDefault == "" {
		apiDefault = "https://api.agilestacks.io"
	}
	RootCmd.PersistentFlags().StringVar(&config.ApiBaseUrl, "api", apiDefault, "Hub API service base URL, HUB_API")

	RootCmd.PersistentFlags().StringVar(&config.AwsProfile, "aws_profile", os.Getenv("AWS_PROFILE"), "AWS ~/.aws/credentials profile, AWS_PROFILE")
	awsRegion := os.Getenv(envVarNameAwsRegion)
	if awsRegion == "" {
		awsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	RootCmd.PersistentFlags().StringVar(&config.AwsRegion, "aws_region", awsRegion, "AWS region hint (for S3 state access), AWS_DEFAULT_REGION")
	RootCmd.PersistentFlags().BoolVar(&config.AwsUseIamRoleCredentials, "aws_use_iam_role_credentials", true, "Try EC2 instance credentials")
	RootCmd.PersistentFlags().BoolVar(&config.AwsPreferProfileCredentials, "aws_prefer_profile_credentials", false, "Try AWS CLI config profile credentials first, before OS env")

	RootCmd.PersistentFlags().StringVar(&config.GcpCredentialsFile, "gcp_credentials_file", "", "Path to GCP Service Account keys JSON file, GOOGLE_APPLICATION_CREDENTIALS, see https://cloud.google.com/docs/authentication/getting-started")
	RootCmd.PersistentFlags().StringVar(&config.AzureCredentialsFile, "azure_credentials_file", "", "Path to Azure Service Principal auth JSON file, AZURE_AUTH_LOCATION, see https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-authorization")

	RootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", true, "Verbose mode")
	RootCmd.PersistentFlags().BoolVarP(&config.Debug, "debug", "d", false, "Print debug info. Or set HUB_DEBUG=1")
	RootCmd.PersistentFlags().BoolVar(&config.Trace, "trace", false, "Print detailed trace info. Or set HUB_TRACE=1")
	RootCmd.PersistentFlags().StringVar(&config.LogDestination, "log-destination", "stderr", "stderr or stdout")
	RootCmd.PersistentFlags().StringVar(&config.TtyMode, "tty", "autodetect", "Terminal mode for colors, etc. true / false")

	RootCmd.PersistentFlags().BoolVar(&config.AggWarnings, "all-warnings", true, "Repeat all warnings before [successful] exit")
	RootCmd.PersistentFlags().BoolVarP(&config.Force, "force", "f", false, "Force operation despite of errors. Or set HUB_FORCE=1")

	RootCmd.PersistentFlags().BoolVar(&config.Compressed, "compressed", true, "Write gzip compressed files")
	RootCmd.PersistentFlags().StringVar(&config.EncryptionMode, "encrypted", "if-key-set",
		"Write encrypted files if HUB_CRYPTO_PASSWORD, HUB_CRYPTO_AWS_KMS_KEY_ARN, HUB_CRYPTO_AZURE_KEYVAULT_KEY_ID is set. true / false")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		util.Warn("Unable to determine HOME directory: %v", err)
	}
	if config.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(config.ConfigFile)
	} else {
		if err == nil {
			// Search config in home directory with name ".hub-config" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigName(".hub-config")
		}
	}
	if config.CacheFile == "" && err == nil {
		config.CacheFile = fmt.Sprintf("%s/.hub-cache.yaml", home)
	}

	viper.SetEnvPrefix("hub")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err = viper.ReadInConfig(); err == nil {
		if config.Verbose {
			log.Printf("Using config file %s", viper.ConfigFileUsed())
		}
	}
	if api := viper.GetString("api"); api != "" {
		config.ApiBaseUrl = api
	}
	if loginToken := viper.GetString("token"); loginToken != "" {
		config.ApiLoginToken = loginToken
	}
	if viper.GetBool("force") {
		config.Force = true
	}
	if viper.GetBool("debug") {
		config.Debug = true
	}
	if viper.GetBool("trace") {
		config.Trace = true
	}
	if pass := viper.GetString("crypto-password"); pass != "" {
		config.CryptoPassword = pass
	}
	if key := viper.GetString("crypto-aws-kms-key-arn"); key != "" {
		config.CryptoAwsKmsKeyArn = key
	}
	if key := viper.GetString("crypto-azure-keyvault-key-id"); key != "" {
		config.CryptoAzureKeyVaultKeyId = key
	}
}
