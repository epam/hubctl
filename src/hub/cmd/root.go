package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"hub/config"
	"hub/util"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "hub",
	Short: "Automation Hub is a lifecycle and stack composition tool",
	Long: `Hub CLI is an interface to Automation Hub.

Automation Hub is a lifecycle and stack composition tool:
- template and stack creation, stack lifecycle;
- stack instance parameters, output variables and status;
- enumeration of templates, stacks, components;
- inventory.`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Update()
		if config.Debug {
			log.Printf("Hub CLI %s %s\n", util.CliVersion, runtime.Version())
		}
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		util.PrintAllWarnings()
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "Config file (default is $HOME/.automation-hub.{yaml,json})")
	RootCmd.PersistentFlags().StringVar(&config.CacheFile, "cache", "", "API cache file (default is $HOME/.automation-hub-cache.yaml)")

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

	RootCmd.PersistentFlags().StringVar(&config.GcpCredentialsFile, "gcp_credentials_file", "", "Path to GCP Service Account keys JSON file, GOOGLE_APPLICATION_CREDENTIALS")

	RootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", true, "Verbose mode")
	RootCmd.PersistentFlags().BoolVarP(&config.Debug, "debug", "d", false, "Print debug info. Or set HUB_DEBUG=1")
	RootCmd.PersistentFlags().BoolVarP(&config.Trace, "trace", "t", false, "Print detailed trace info. Or set HUB_TRACE=1")
	RootCmd.PersistentFlags().StringVar(&config.LogDestination, "log-destination", "stderr", "stderr or stdout")

	RootCmd.PersistentFlags().BoolVar(&config.AggWarnings, "all-warnings", true, "Repeat all warnings before [successful] exit")
	RootCmd.PersistentFlags().BoolVarP(&config.Force, "force", "f", false, "Force operation despite of errors. Or set HUB_FORCE=1")

	RootCmd.PersistentFlags().BoolVar(&config.Compressed, "compressed", true, "Write gzip compressed files")
	RootCmd.PersistentFlags().StringVar(&config.EncryptionMode, "encrypted", "if-password-set",
		"Write encrypted files if HUB_CRYPTO_PASSWORD is set. true / false")
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
			// Search config in home directory with name ".automation-hub" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigName(".automation-hub")
		}
	}
	if config.CacheFile == "" && err == nil {
		config.CacheFile = fmt.Sprintf("%s/.automation-hub-cache.yaml", home)
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
}
