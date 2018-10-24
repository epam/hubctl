package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"hub/git"
)

var (
	reset              bool
	recurse            bool
	optimizeGitRemotes bool
)

var pullCmd = &cobra.Command{
	Use:   "pull hub.yaml [-b <base directory>] [-f] [-r]",
	Short: "Pull stack sources",
	Long:  `Clone or update stack and component sources from Git.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pull(args)
	},
}

func pull(args []string) error {
	if len(args) != 1 {
		return errors.New("Pull command has only one argument - path to Stack Manifest file")
	}

	manifest := args[0]
	git.Pull(manifest, componentsBaseDir, reset, recurse, optimizeGitRemotes)

	return nil
}

func init() {
	pullCmd.Flags().StringVarP(&componentsBaseDir, "baseDir", "b", "",
		"Path to base directory to clone sources into (default to manifest dir)")
	pullCmd.Flags().BoolVarP(&optimizeGitRemotes, "optimize-git-remotes", "", true,
		"Optimize Git remote with local clone (same remote repository is encountered more than once)")
	pullCmd.Flags().BoolVarP(&reset, "reset", "r", false,
		"Hard-reset Git tree prior to update (kinda broken?)")
	pullCmd.Flags().BoolVarP(&recurse, "recurse", "s", true,
		"Recurse into fromStack")
	RootCmd.AddCommand(pullCmd)
}
