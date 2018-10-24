package compose

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/state"
	"hub/storage"
	"hub/util"
)

func BackupUnbundle(bundlesFilenames [][]string, parametersFiles []string,
	rename, erase, includeOnly []string) {

	renames := prepareComponentRenames(rename)

	if len(parametersFiles) == 0 && config.Verbose && !config.Debug {
		config.Verbose = false
		config.AggWarnings = false
	}

	if config.Verbose {
		params := ""
		if len(parametersFiles) > 0 {
			params = fmt.Sprintf(" into %v", parametersFiles)
		}
		log.Printf("Unbundling %v%s", bundlesFilenames, params)
	}

	bundles := parseBundles(bundlesFilenames)

	var outputs *storage.Files
	if len(parametersFiles) > 0 {
		checked, errs := storage.Check(parametersFiles, "restore parameters")
		if len(errs) > 0 {
			if config.Force {
				util.Warn("Unable to check output parameters file(s): %v", util.Errors2(errs...))
			} else {
				log.Fatalf("Unable to check output parameters file(s): %v", util.Errors2(errs...))
			}
		}
		outputs = checked
	}

	allBackups := extractBackups(bundles, renames, erase, includeOnly)
	backups := selectBackups(allBackups)
	warnBackupPriority(backups, allBackups)
	params := transformBackupsToParameters(backups)
	yamlBytes, err := yaml.Marshal(params)
	if err != nil {
		log.Fatalf("Unable to marshal parameters into YAML: %v", err)
	}
	if outputs != nil {
		storage.Write(yamlBytes, outputs)
	} else {
		os.Stdout.Write([]byte("--- yaml\n"))
		os.Stdout.Write(yamlBytes)
	}
}

func prepareComponentRenames(rename []string) map[string]string {
	renames := make(map[string]string)
	for _, ren := range rename {
		fromTo := strings.Split(ren, ":")
		if len(fromTo) != 2 {
			log.Fatalf("Bad component rename specification `%s`", ren)
		}
		from := fromTo[0]
		to := fromTo[1]
		if from == "" || to == "" {
			log.Fatalf("Bad component rename specification `%s`", ren)
		}
		toAlready, exist := renames[from]
		if exist {
			log.Fatalf("`%s` already renamed to `%s`, cannot rename to `%s`", from, toAlready, to)
		}
	}
	return renames
}

func parseBundles(bundlesFilenames [][]string) []*state.BackupManifest {
	bundles := make([]*state.BackupManifest, 0, len(bundlesFilenames))
	for _, filename := range bundlesFilenames {
		bundles = append(bundles, state.MustParseBackupBundles(filename))
	}
	return bundles
}

func extractBackups(bundles []*state.BackupManifest,
	renames map[string]string, erase, includeOnly []string) map[string][]state.ComponentBackup {

	allBackups := make(map[string][]state.ComponentBackup)
	for fileIndex, bundle := range bundles {
		for componentName, backup := range bundle.Components {
			if util.Contains(erase, componentName) {
				continue
			}
			if len(includeOnly) > 0 && !util.Contains(includeOnly, componentName) {
				continue
			}

			renamedTo, renamed := renames[componentName]
			if renamed {
				componentName = renamedTo
			}

			prev, exist := allBackups[componentName]
			backup.Source = bundle.Source
			backup.FileIndex = fileIndex
			if !exist {
				allBackups[componentName] = []state.ComponentBackup{backup}
			} else {
				allBackups[componentName] = append(prev, backup)
			}
		}
	}
	return allBackups
}

func selectBackups(allBackups map[string][]state.ComponentBackup) map[string]state.ComponentBackup {
	backups := make(map[string]state.ComponentBackup)
	for componentName, componentBackups := range allBackups {
		var i int
		for i = len(componentBackups) - 1; i >= 0; i-- {
			backup := componentBackups[i]
			if backup.Status == "success" {
				backups[componentName] = backup
				break
			}
		}
		if i < 0 {
			util.Warn("No valid backup found for component `%s`", componentName)
		}
	}
	return backups
}

func warnBackupPriority(backups map[string]state.ComponentBackup, allBackups map[string][]state.ComponentBackup) {
	for componentName, componentBackup := range backups {
		others := allBackups[componentName]
		for i := len(others) - 1; i >= 0; i-- {
			other := others[i]
			if componentBackup.Source != other.Source {
				if componentBackup.FileIndex < other.FileIndex && other.Status == "error" {
					util.Warn("Component `%s` backup from `%s` (status `%s`) is used instead of backup from `%s` (status `%s`)",
						componentName,
						componentBackup.Source, componentBackup.Status,
						other.Source, other.Status)
				}
				if componentBackup.Timestamp.Before(other.Timestamp) && componentBackup.FileIndex > other.FileIndex {
					util.Warn("Component `%s` backup from `%s` (timestamp `%s`) is used instead of backup from `%s` (timestamp `%s`)",
						componentName,
						componentBackup.Source, componentBackup.Timestamp.String(),
						other.Source, other.Timestamp.String())
				}
			}
		}
	}
}

func transformBackupsToParameters(backups map[string]state.ComponentBackup) *manifest.ParametersManifest {
	params := make([]manifest.Parameter, 0, len(backups))
	for componentName, componentBackup := range backups {
		// TODO find a common root and output a parameters sub-tree with single `component:` declaration
		for _, output := range componentBackup.Outputs {
			params = append(params, manifest.Parameter{
				Component: componentName,
				Name:      output.Name,
				Value:     output.Value,
			})
		}
	}
	return &manifest.ParametersManifest{Parameters: params}
}
