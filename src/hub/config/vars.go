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

	AwsProfile               string
	AwsRegion                string
	AwsUseIamRoleCredentials bool

	Verbose bool
	Debug   bool
	Trace   bool

	LogDestination string

	AggWarnings             bool
	Force                   bool
	SwitchKubeconfigContext bool
	Compressed              bool
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
}
