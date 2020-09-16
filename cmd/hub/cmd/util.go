package cmd

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/lifecycle"
	"github.com/agilestacks/hub/cmd/hub/metrics"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var utilCmd = &cobra.Command{
	Use:   "util <otp | ...>",
	Short: "Utility functions",
}

var utilOtpCmd = &cobra.Command{
	Use:   "otp [encode]",
	Short: "Encode stdin with one-time pad",
	Long: `Encode stdin with one-time pad provided via HUB_RANDOM environment variable (base64).

The result is printed to stdout:

	secrets = <base64-encoded-result>

HUB_RANDOM is set by deploy command for components to output secrets securely.
In component's Makefile:

	@echo
	@echo Outputs:
	@echo "password = $(password)\ntoken = $(token)" | hub util otp
	@echo

Do not call "hub util otp" more than once from the same component.
https://en.wikipedia.org/wiki/One-time_pad
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return otpEncode(args)
	},
}

var utilMetricsCmd = &cobra.Command{
	Use:   "metrics <command>",
	Short: "Send usage metrics",
	Long: `Send usage metrics in background to SuperHub and Datadog.

We value your privacy and only send anonymized usage metrics for following commands:
- elaborate
- deploy
- undeploy
- backup create
- api *

Usage metric contain:
- Hub CLI command invoked without arguments, ie. 'deploy' or 'backup create', or 'api instance get'
- synthetic machine id - an UUID generated in first interactive session (stdout is a TTY)
- usage counter - 1 per invocation

Edit $HOME/.hub-cache.yaml to change settings:

  metrics:
    disabled: false
    host: 68af657e-6a51-4d4b-890c-4b548852724d

Set 'disabled: true' to skip usage metrics reporting.
Set 'host: ""' to send the counter but not the UUID.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return putMetrics(args)
	},
}

func otpEncode(args []string) error {
	if len(args) != 1 && len(args) != 0 || (len(args) == 1 && args[0] != "encode") {
		return errors.New("OTP command has only one optional argument - [encode]")
	}

	base64Random := os.Getenv(lifecycle.HubEnvVarNameRandom)
	if base64Random == "" {
		return fmt.Errorf("%s is not set", lifecycle.HubEnvVarNameRandom)
	}
	random, err := base64.RawStdEncoding.DecodeString(base64Random)
	if err != nil {
		log.Fatalf("Unable to decode base64 random: %v", err)
	}

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Unable to read input (read %d bytes): %v", len(input), err)
	}
	if len(input) == 0 || len(bytes.Trim(input, " \n\r")) == 0 {
		return nil
	}

	output, err := util.OtpEncode(input, random)
	if err != nil {
		log.Fatalf("Unable to encode one-time pad: %v", err)
	}
	fmt.Printf("secrets = %s\n", output)

	return nil
}

func putMetrics(args []string) error {
	if len(args) != 1 {
		return errors.New("Metrics command has only one argument - command to send usage metric for")
	}
	cmd := args[0]

	metrics.PutMetrics(cmd)

	return nil
}

func init() {
	utilCmd.AddCommand(utilOtpCmd)
	utilCmd.AddCommand(utilMetricsCmd)
	RootCmd.AddCommand(utilCmd)
}
