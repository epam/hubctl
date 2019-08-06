package config

import (
	"log"
	"os"
)

var (
	ConfigFile string
	CacheFile  string

	ApiBaseUrl    string
	ApiLoginToken string

	AwsProfile                  string
	AwsRegion                   string
	AwsPreferProfileCredentials bool
	AwsUseIamRoleCredentials    bool
	GcpCredentialsFile          string
	AzureCredentialsFile        string

	Verbose bool
	Debug   bool
	Trace   bool

	LogDestination string

	AggWarnings             bool
	Force                   bool
	SwitchKubeconfigContext bool
	Compressed              bool
	Encrypted               bool
	EncryptionMode          string
	CryptoPassword          string
)

func Update() {
	if LogDestination == "stdout" {
		log.SetOutput(os.Stdout)
	} else if LogDestination != "stderr" {
		log.Fatalf("Unknown --log-destination `%s`", LogDestination)
	}
	if Trace {
		Debug = true
	}
	if Debug {
		Verbose = true
	}
	if Force {
		log.Print("Force flag set, some errors will be treated as warnings")
	}
	switch EncryptionMode {
	case "true":
		if CryptoPassword == "" {
			log.Fatal("Set HUB_CRYPTO_PASSWORD='random password' for --encrypted=true")
		}
		Encrypted = true
	case "false":
		Encrypted = false
	case "if-password-set":
		Encrypted = CryptoPassword != ""
	default:
		log.Fatalf("Unknown --encrypted `%s`", EncryptionMode)
	}
}
