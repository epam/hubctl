package cmd

import (
	"errors"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docCmd = &cobra.Command{
	Use:   "doc <directory>",
	Short: "Generate Hub CLI reference in Markdown format",
	Long:  `Generate command reference and store files under directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateDoc(args)
	},
}

func generateDoc(args []string) error {
	if len(args) != 1 {
		return errors.New("doc command has one mandatory argument - path to directory to store generated Markdown files")
	}

	dir := args[0]

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(dir, 0775)
			if err != nil {
				log.Fatalf("Unable to create `%s`: %v", dir, err)
			}
		} else {
			log.Fatalf("Unable to stat `%s`: %v", dir, err)
		}
	} else {
		if info != nil && !info.IsDir() {
			log.Fatalf("`%s` is not a directory", dir)
		}
	}

	err = doc.GenMarkdownTree(RootCmd, dir)
	if err != nil {
		log.Fatalf("Unable to generate command reference: %v", err)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(docCmd)
}
