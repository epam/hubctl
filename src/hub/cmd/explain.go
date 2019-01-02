package cmd

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"hub/state"
	"hub/util"
)

var (
	explainGlobal bool
	explainRaw    bool
	explainInKv   bool
	explainInSh   bool
	explainInJson bool
	explainInYaml bool
	explainColor  bool
)

var explainCmd = &cobra.Command{
	Use:   "explain [hub.yaml.elaborate] hub.yaml.state[,s3://bucket/hub.yaml.state]",
	Short: "Explain stack outputs, provides, and parameters evolution",
	Long: `Display user-level stack outputs, history of parameters evolution during deployment,
and component's capabilities. Parameters history is read from state file. Elaborate file is optional.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return explain(args)
	},
}

func explain(args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return errors.New("Explain command has two arguments - path to Stack Elaborate file (optional) and to State file")
	}

	elaborateManifests := []string{}
	i := 0
	if len(args) == 2 {
		elaborateManifests = util.SplitPaths(args[0])
		i = 1
	}
	stateManifests := util.SplitPaths(args[i])

	format := "text"
	if explainInKv {
		format = "kv"
	} else if explainInSh {
		format = "sh"
	} else if explainInJson {
		format = "json"
	} else if explainInYaml {
		format = "yaml"
	}

	state.Explain(elaborateManifests, stateManifests, explainGlobal, componentName, explainRaw, format, explainColor)

	return nil
}

func init() {
	explainCmd.Flags().BoolVarP(&explainGlobal, "global", "g", false,
		"Display Stack or Application parameters and outputs")
	explainCmd.Flags().StringVarP(&componentName, "component", "c", "",
		"Component to explain")
	explainCmd.Flags().BoolVarP(&explainRaw, "raw-outputs", "r", false,
		"Display raw component outputs")
	explainCmd.Flags().BoolVarP(&explainInKv, "kv", "", false,
		"key=value output")
	explainCmd.Flags().BoolVarP(&explainInSh, "sh", "", false,
		"Shell output")
	explainCmd.Flags().BoolVarP(&explainInJson, "json", "", false,
		"JSON output")
	explainCmd.Flags().BoolVarP(&explainInYaml, "yaml", "", false,
		"YAML output")
	explainCmd.Flags().BoolVarP(&explainColor, "color", "", isatty.IsTerminal(os.Stdout.Fd()),
		"Colorized output")
	RootCmd.AddCommand(explainCmd)
}
