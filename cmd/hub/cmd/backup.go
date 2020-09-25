package cmd

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/compose"
	"github.com/agilestacks/hub/cmd/hub/lifecycle"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	backupBundleInJson          bool
	backupAllowPartial          bool
	backupRenameComponents      string
	backupEraseComponents       string
	backupIncludeOnlyComponents string
)

var backupCmd = &cobra.Command{
	Use:   "backup <create | unbundle> ...",
	Short: "Create and manage backups",
	Long:  `Create backup of stack components; transform backup bundle into parameters manifest.`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create hub.yaml.elaborate -s hub.yaml.state[,s3://bucket/hub.yaml.state] -o bundle.yaml",
	Short: "Create backup bundle",
	Long: `Create backup of stack component(s).
Each stack component that supports 'backup' verb is invoked.
Bundle can be saved into multiple files and also sent to S3.`,
	Annotations: map[string]string{
		"usage-metering": "tags",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		pipe := cmdContextPipe(cmd)
		if pipe != nil {
			defer pipe.Close()
		}
		return backupCreate(args, pipe)
	},
}

var backupUnbundleCmd = &cobra.Command{
	Use:   "unbundle bundle.yaml [bundle2.yaml ...] -o restore-params.yaml[,s3://bucket/params.yaml]",
	Short: "Create parameters for restore from bundle",
	Long: `Transform backup bundle(s) into parameters manifest.
Multiple bundles will be merged with priority determined by file order on command-line (from left to right).
Components can be renamed, included and excluded.
Parameters can be saved into multiple files and also sent to S3.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return backupUnbundle(args)
	},
}

func backupCreate(args []string, pipe io.WriteCloser) error {
	if len(args) != 1 {
		return errors.New("Backup Create command has only one argument - path to Stack Elaborate file")
	}

	manifests := util.SplitPaths(args[0])
	stateManifests := util.SplitPaths(stateManifestExplicit)
	bundleFiles := util.SplitPaths(outputFiles)
	components := util.SplitPaths(componentName)

	setOsEnvForNestedCli(manifests, stateManifests, componentsBaseDir)

	request := &lifecycle.Request{
		Verb:                 "backup",
		DryRun:               dryRun,
		ManifestFilenames:    manifests,
		StateFilenames:       stateManifests,
		Components:           components,
		OsEnvironmentMode:    osEnvironmentMode,
		ComponentsBaseDir:    componentsBaseDir,
		PipeOutputInRealtime: pipeOutputInRealtime,
		Environment:          hubEnvironment,
		StackInstance:        hubStackInstance,
		Application:          hubApplication,
	}

	lifecycle.BackupCreate(request, bundleFiles, backupBundleInJson, backupAllowPartial, pipe)

	return nil
}

func backupUnbundle(args []string) error {
	if len(args) == 0 {
		return errors.New("Backup Unbunle command has one or more arguments - path(s) to Backup Bundle file(s)")
	}

	bundles := make([][]string, 0, len(args))
	for _, files := range args {
		bundles = append(bundles, strings.Split(files, ","))
	}
	parametersFiles := util.SplitPaths(outputFiles)
	rename := util.SplitPaths(backupRenameComponents)
	erase := util.SplitPaths(backupEraseComponents)
	includeOnly := util.SplitPaths(backupIncludeOnlyComponents)

	compose.BackupUnbundle(bundles, parametersFiles,
		rename, erase, includeOnly)

	return nil
}

var apiBackupCmd = &cobra.Command{
	Use:   "backup <get | delete> ...",
	Short: "List and manage Stack Instance backups",
}

var apiBackupGetCmd = &cobra.Command{
	Use:   "get [id | name]",
	Short: "Show a list of Backups or details about the Backup",
	Long: `Show a list of all user accessible Backups or details about
the particular Backup (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return apiBackup(args)
	},
}

var apiBackupDeleteCmd = &cobra.Command{
	Use:   "delete <id | name>",
	Short: "Delete Backup by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteApiBackup(args)
	},
}

func apiBackup(args []string) error {
	if len(args) > 1 {
		return errors.New("Backup command has one optional argument - id or name of the backup")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Backups(selector, showLogs, jsonFormat)

	return nil
}

func deleteApiBackup(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Backup command has one mandatory argument - id or name of the backup")
	}

	api.DeleteBackup(args[0])

	return nil
}

func init() {
	backupCreateCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		"Path to state file(s), for example hub.yaml.state,s3://bucket/hub.yaml.state")
	backupCreateCmd.Flags().StringVarP(&outputFiles, "output", "o", "",
		"Bundle output file(s), for example bundle.yaml,s3://bucket/bundle.yaml (default to stdout)")
	backupCreateCmd.Flags().BoolVarP(&backupBundleInJson, "json", "", false,
		"JSON output")
	backupCreateCmd.Flags().StringVarP(&componentName, "components", "c", "",
		"A list of components to backup (in order, separated by comma)")
	backupCreateCmd.Flags().BoolVarP(&backupAllowPartial, "allow-partial", "", false,
		"Allow partial backups to succeed")
	initCommonLifecycleFlags(backupCreateCmd, "backup")

	backupUnbundleCmd.Flags().StringVarP(&outputFiles, "output", "o", "",
		"Parameters output file(s), optionally write to S3 (default to stdout)")
	backupUnbundleCmd.Flags().StringVarP(&backupRenameComponents, "rename", "r", "",
		"Components to rename, for example -r pg1:postgresql,pg2:postgresql-rds")
	backupUnbundleCmd.Flags().StringVarP(&backupEraseComponents, "erase", "e", "",
		"Components to omit from parameters file, for example -e etcd,vault")
	backupUnbundleCmd.Flags().StringVarP(&backupIncludeOnlyComponents, "include-only", "i", "",
		"Include only specified components, for example -i postgresql,postgresql-rds")

	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupUnbundleCmd)
	RootCmd.AddCommand(backupCmd)

	apiBackupGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	apiBackupGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	apiBackupCmd.AddCommand(apiBackupGetCmd)
	apiBackupCmd.AddCommand(apiBackupDeleteCmd)
	apiCmd.AddCommand(apiBackupCmd)
}
