// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/epam/hubctl/cmd/hub/util"
)

func PullManifest(manifestFilename string, baseDir string, reset, recurse, optimizeGitRemotes, asSubtree bool) {

	components, repos, manifests := pullManifest(manifestFilename, baseDir, reset, recurse, optimizeGitRemotes, asSubtree,
		make([]string, 0), make([]LocalGitRepo, 0), make([]string, 0))

	if len(repos) == 0 {
		log.Printf("No Git sources found in %s", strings.Join(manifests, ", "))
	} else if config.Verbose {
		log.Printf("Components sourced from Git: %s", strings.Join(components, ", "))
		if config.Debug {
			printLocalGitRepos(repos)
		}
	}
}

func pullManifest(manifestFilename string, baseDir string, reset, recurse, optimizeGitRemotes, asSubtree bool,
	components []string, repos []LocalGitRepo, manifests []string) ([]string, []LocalGitRepo, []string) {

	stackManifest, rest, _, err := manifest.ParseManifest([]string{manifestFilename})
	if err != nil {
		log.Fatalf("Unable to pull %s: %v", manifestFilename, err)
	}
	if len(rest) > 0 {
		log.Printf("Stack manifest %s contains multiple YAML documents - using first document only", manifestFilename)
	}

	baseDirCurrent := baseDir
	stackBaseDir := util.StripDotDirs(filepath.Dir(manifestFilename))
	if baseDirCurrent == "" {
		baseDirCurrent = stackBaseDir
	}
	if config.Debug {
		log.Printf("Base directory for sources is `%s`", baseDirCurrent)
	}

	order, err := manifest.GenerateLifecycleOrder(stackManifest)
	if err != nil {
		log.Fatal(err)
	}
	stackManifest.Lifecycle.Order = order

	stackName := stackManifest.Meta.Name
	if i := strings.Index(stackName, ":"); i > 0 {
		stackName = stackName[0:i]
	}

	components, repos, err = getGit(stackManifest.Meta.Source.Git, baseDirCurrent, stackName,
		stackName, reset, optimizeGitRemotes, false, components, repos)
	if err != nil {
		log.Fatalf("%v", err)
	}
	for _, component := range stackManifest.Components {
		components, repos, err = getGit(component.Source.Git,
			baseDirCurrent, manifest.ComponentSourceDirFromRef(&component, stackBaseDir, baseDirCurrent), // TODO proper dir
			manifest.ComponentQualifiedNameFromRef(&component),
			reset, optimizeGitRemotes, asSubtree,
			components, repos)
		if err != nil {
			if config.Force {
				util.Warn("%v", err)
			} else {
				log.Fatalf("%v", err)
			}
		}
	}

	manifests = append(manifests, manifestFilename)

	if recurse && stackManifest.Meta.FromStack != "" {
		fromStackManifestFilename := filepath.Join(stackManifest.Meta.FromStack, "hub.yaml")
		if config.Debug {
			log.Printf("Recursing into %s", fromStackManifestFilename)
		}
		components, repos, manifests = pullManifest(fromStackManifestFilename, baseDir, reset, recurse, optimizeGitRemotes, asSubtree,
			components, repos, manifests)
	}

	return components, repos, manifests
}

func getGit(source manifest.Git, baseDir string, relDir string, componentName string, reset, optimizeGitRemotes, asSubtree bool,
	components []string, repos []LocalGitRepo) ([]string, []LocalGitRepo, error) {

	if source.Remote == "" || util.Contains(components, componentName) {
		return components, repos, nil
	}

	if source.LocalDir != "" {
		relDir = source.LocalDir
	}

	dir := relDir
	if !filepath.IsAbs(relDir) {
		relDir = filepath.Join(baseDir, relDir)
		var err error
		dir, err = filepath.Abs(relDir)
		if err != nil {
			return components, repos,
				fmt.Errorf("Error determining absolute path to pull into %s: %v", relDir, err)
		}
	}
	if config.Debug {
		log.Printf("Component `%s` Git repo dir is `%s`", componentName, dir)
	}
	if dirInRepoList(dir, repos) {
		return components, repos,
			fmt.Errorf("Directory %s used twice to pull Git repo", dir)
	}

	clone, err := emptyDir(dir, !asSubtree)
	if err != nil {
		return components, repos, err
	}

	remote := source.Remote
	remoteVerbose := fmt.Sprintf("`%s`", source.Remote)
	if optimizeGitRemotes && maybeRemote(remote) {
		remote = findLocalClone(repos, source.Remote, source.Ref)
		if remote != source.Remote {
			remoteVerbose = fmt.Sprintf("`%s` (%s)", remote, source.Remote)
			if config.Debug {
				log.Printf("Optimized component `%s` origin from `%s` to `%s`", componentName, source.Remote, remote)
			}
		}
	}

	if clone {
		if config.Verbose {
			log.Printf("Cloning from %s", remoteVerbose)
		}
		if asSubtree {
			return components, repos, errors.New("not implemented")
		} else {
			err = Clone(remote, source.Ref, dir)
			if err != nil {
				return components, repos, err
			}
		}
	} else {
		if config.Verbose {
			log.Printf("Updating from %s", remoteVerbose)
		}
		if asSubtree {
			// ensure remote with name = remote-<component name>
			// fetch source.Ref as _remote-<component name>/<Ref> remote branch
			// remember current branch
			// checkout _remote-<component name>/<Ref> as _remote-<component name>-<Ref>
			// split source.SubDir into _split-<component name>
			// pop to current branch
			// subtree merge into `dir` from _split-<component name>
			// delete _split-<component name>
			// delete _remote-<component name>-<Ref>
			// in case of error - show error, then note user to:
			// - return to current branch
			// - delete _split and _remote branches
			return components, repos, errors.New("not implemented")
		} else {
			if reset {
				// 	cmd := exec.Cmd{
				// 		Path: gitBin,
				// 		Dir:  dir,
				// 		Args: []string{"git", "stash", "--include-untracked"},
				// 	}
				// 	gitDebug(&cmd)
				// 	err = cmd.Run()
				// 	if err != nil {
				// 		return components, repos,
				// 			fmt.Errorf("Unable to stash Git repo worktree `%s`: %v", dir, err)
				// 	}
				return components, repos, errors.New("not implemented")
			}

			err = Pull(source.Ref, dir)
			if err != nil {
				return components, repos,
					fmt.Errorf("Unable to pull Git repo %s into `%s`: %v", remoteVerbose, dir, err)
			}
		}
	}

	headName, _, err := HeadInfo(dir)
	if err != nil {
		util.Warn("%v", err)
	}

	return append(components, componentName),
		append(repos, LocalGitRepo{
			Remote:          source.Remote,
			OptimizedRemote: remote,
			Ref:             source.Ref,
			HeadRef:         headName,
			SubDir:          source.SubDir,
			AbsDir:          dir,
		}),
		nil
}

var dirMode = os.FileMode(0755)

func emptyDir(dir string, removeContentIfForced bool) (bool, error) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if !util.NoSuchFile(err) {
			return false, fmt.Errorf("Unable to stat `%s`: %v", dir, err)
		}
		return true, nil
	}
	if !dirInfo.IsDir() {
		if config.Force {
			err = os.Remove(dir)
			if err != nil {
				return false, fmt.Errorf("Unable to force remove `%s`: %v", dir, err)
			}
		} else {
			return false, fmt.Errorf("Pull target `%s` is not a directory, add -f / --force to override", dir)
		}
	}
	gitDir := filepath.Join(dir, ".git")
	gitInfo, err := os.Stat(gitDir)
	if err != nil {
		if !util.NoSuchFile(err) {
			return false, fmt.Errorf("Unable to stat `%s`: %v", dir, err)
		}
		dirFD, err := os.Open(dir)
		if err != nil {
			return false, fmt.Errorf("Unable to open dir `%s`: %v", dir, err)
		}
		fileNames, err := dirFD.Readdirnames(1)
		if err != nil && err != io.EOF {
			return false, fmt.Errorf("Unable to read dir `%s`: %v", dir, err)
		}
		if config.Trace {
			log.Printf("Dir content: %v", fileNames)
		}
		if len(fileNames) > 0 {
			if !removeContentIfForced {
				return false, nil
			}
			if config.Force {
				err := os.RemoveAll(dir)
				if err != nil {
					return false, fmt.Errorf("Unable to force clean dir `%s`: %v", dir, err)
				}
				os.Mkdir(dir, dirMode)
			} else {
				return false, fmt.Errorf("Pull target `%s` is not an empty directory, add -f / --force to override", dir)
			}
		}
		dirFD.Close()
		return true, nil
	} else {
		if !gitInfo.IsDir() {
			return false, fmt.Errorf("Pull target `%s` is not a Git repo", dir)
		}
	}
	return false, nil
}

func maybeRemote(origin string) bool {
	return strings.Contains(origin, ":")
}

func findLocalClone(repos []LocalGitRepo, remote string, ref string) string {
	for _, repo := range repos {
		if remote == repo.Remote && ref == repo.Ref {
			return repo.AbsDir
		}
	}
	return remote
}

// func upToDate(err error) bool {
// 	return strings.Contains(err.Error(), "already up-to-date")
// }

func dirInRepoList(dir string, repos []LocalGitRepo) bool {
	for _, repo := range repos {
		if dir == repo.AbsDir {
			return true
		}
	}
	return false
}
