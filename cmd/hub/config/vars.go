package config

import (
	"log"
	"os"

	"github.com/mattn/go-isatty"
)

var (
	ConfigFile string
	CacheFile  string

	ApiBaseUrl      string
	ApiLoginToken   string
	ApiDerefSecrets bool
	ApiTimeout      int = 30

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
	TtyMode        string
	Tty            bool
	TtyForced      bool

	AggWarnings             bool
	Force                   bool
	SwitchKubeconfigContext bool
	Compressed              bool
	Encrypted               bool
	EncryptionMode          string

	CryptoPassword           string
	CryptoAwsKmsKeyArn       string
	CryptoAzureKeyVaultKeyId string

	GitBinDefault = "/usr/bin/git"
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
		if CryptoPassword == "" && CryptoAwsKmsKeyArn == "" && CryptoAzureKeyVaultKeyId == "" {
			log.Fatal("For --encrypted=true, set HUB_CRYPTO_PASSWORD='random password' or HUB_CRYPTO_AWS_KMS_KEY_ARN='arn:aws:kms:...' or HUB_CRYPTO_AZURE_KEYVAULT_KEY_ID='https://*.vault.azure.net/keys/...'")
		}
		Encrypted = true
	case "false":
		Encrypted = false
	case "if-key-set":
		Encrypted = CryptoPassword != "" || CryptoAwsKmsKeyArn != "" || CryptoAzureKeyVaultKeyId != ""
	default:
		log.Fatalf("Unknown --encrypted `%s`", EncryptionMode)
	}

	tty := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsTerminal(os.Stderr.Fd())
	switch TtyMode {
	case "true":
		Tty = true
		TtyForced = !tty
	case "false":
		Tty = false
	case "autodetect":
		Tty = tty
	default:
		log.Fatalf("Unknown --tty `%s`", TtyMode)
	}
}
