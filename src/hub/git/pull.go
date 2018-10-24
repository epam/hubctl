package git

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"hub/config"
	"hub/manifest"
	"hub/util"
)

func Pull(manifestFilename string, baseDir string, reset, recurse, optimizeGitRemotes bool) {

	components, repos, manifests := pull(manifestFilename, baseDir, reset, recurse, optimizeGitRemotes,
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

func pull(manifestFilename string, baseDir string, reset, recurse, optimizeGitRemotes bool,
	components []string, repos []LocalGitRepo, manifests []string) ([]string, []LocalGitRepo, []string) {

	stackManifest, rest, _, err := manifest.ParseManifest([]string{manifestFilename})
	if err != nil {
		log.Fatalf("Unable to pull %s: %v", manifestFilename, err)
	}
	if len(rest) > 0 {
		log.Printf("Stack manifest %s contains multiple YAML documents - using first document only", manifestFilename)
	}

	baseDirCurrent := baseDir
	if baseDirCurrent == "" {
		baseDirCurrent = util.StripDotDirs(filepath.Dir(manifestFilename))
	}
	if config.Debug {
		log.Printf("Base directory for sources is `%s`", baseDirCurrent)
	}

	stackName := stackManifest.Meta.Name
	if i := strings.Index(stackName, ":"); i > 0 {
		stackName = stackName[0:i]
	}

	components, repos = getGit(stackManifest.Meta.Source.Git, baseDirCurrent, stackName,
		stackName, reset, optimizeGitRemotes, components, repos)
	for _, component := range stackManifest.Components {
		components, repos = getGit(component.Source.Git, baseDirCurrent, manifest.ComponentSourceDirNameFromRef(&component),
			manifest.ComponentQualifiedNameFromRef(&component), reset, optimizeGitRemotes, components, repos)
	}

	manifests = append(manifests, manifestFilename)

	if recurse && stackManifest.Meta.FromStack != "" {
		fromStackManifestFilename := filepath.Join(stackManifest.Meta.FromStack, "hub.yaml")
		if config.Debug {
			log.Printf("Recursing into %s", fromStackManifestFilename)
		}
		components, repos, manifests = pull(fromStackManifestFilename, baseDir, reset, recurse, optimizeGitRemotes,
			components, repos, manifests)
	}

	return components, repos, manifests
}

func getGit(source manifest.Git, baseDir string, relDir string, componentName string, reset, optimizeGitRemotes bool,
	components []string, repos []LocalGitRepo) ([]string, []LocalGitRepo) {

	if source.Remote == "" || componentExistInList(componentName, components) {
		return components, repos
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
			log.Fatalf("Error determining absolute path to pull into %s: %v", relDir, err)
		}
	}
	if config.Debug {
		log.Printf("Component `%s` Git repo dir is `%s`", componentName, dir)
	}
	if dirExistInList(dir, repos) {
		log.Fatalf("Directory %s used twice to pull Git repo", dir)
	}

	clone := emptyDir(dir)

	progress := os.Stdout
	if !config.Verbose {
		progress = nil
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

	var repo *git.Repository
	var err error
	if clone {
		if config.Verbose {
			log.Printf("Cloning from %s", remoteVerbose)
		}
		repo, err = git.PlainClone(dir, false, &git.CloneOptions{
			URL:           remote,
			ReferenceName: plumbing.ReferenceName(source.Ref),
			SingleBranch:  true,
			Depth:         1,
			Progress:      progress,
		})
		if err != nil {
			log.Fatalf("Unable to clone Git repo %s at `%s` into `%s`: %v", remoteVerbose, source.Ref, dir, err)
		}
	} else {
		if config.Verbose {
			log.Printf("Updating from %s", remoteVerbose)
		}
		repo, err = git.PlainOpen(dir)
		if err != nil {
			log.Fatalf("Unable to open Git repo `%s`: %v", dir, err)
		}
		worktree, err := repo.Worktree()
		if err != nil {
			log.Fatalf("Unable to open Git repo worktree `%s`: %v", dir, err)
		}
		if reset {
			err = worktree.Reset(&git.ResetOptions{Mode: git.HardReset})
			if err != nil {
				log.Fatalf("Unable to hard-reset Git repo worktree `%s`: %v", dir, err)
			}
		}
		err = worktree.Pull(&git.PullOptions{
			ReferenceName: plumbing.ReferenceName(source.Ref),
			Progress:      progress,
		})
		if err != nil && !upToDate(err) {
			log.Fatalf("Unable to pull Git repo %s into `%s`: %v", remoteVerbose, dir, err)
		}
	}
	ref, err := repo.Head()
	if err != nil {
		log.Fatalf("Unable to determine Git repo `%s` HEAD ref: %v", dir, err)
	}

	return append(components, componentName),
		append(repos, LocalGitRepo{
			Remote:          source.Remote,
			OptimizedRemote: remote,
			Ref:             source.Ref,
			HeadRef:         ref.String(),
			SubDir:          source.SubDir,
			AbsDir:          dir,
		})
}

func emptyDir(dir string) bool {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if !util.NoSuchFile(err) {
			log.Fatalf("Unable to stat `%s`: %v", dir, err)
		}
		return true
	}
	if !dirInfo.IsDir() {
		if config.Force {
			err := os.Remove(dir)
			if err != nil {
				log.Fatalf("Unable to force remove `%s`: %v", dir, err)
			}
		} else {
			log.Fatalf("Pull target `%s` is not a directory, add --force to override", dir)
		}
	}
	gitDir := filepath.Join(dir, ".git")
	gitInfo, err := os.Stat(gitDir)
	if err != nil {
		if !util.NoSuchFile(err) {
			log.Fatalf("Unable to stat `%s`: %v", dir, err)
		}
		dirFD, err := os.Open(dir)
		if err != nil {
			log.Fatalf("Unable to open dir `%s`: %v", dir, err)
		}
		fileNames, err := dirFD.Readdirnames(1)
		if err != nil && err != io.EOF {
			log.Fatalf("Unable to read dir `%s`: %v", dir, err)
		}
		if config.Trace {
			log.Printf("Dir content: %v", fileNames)
		}
		if len(fileNames) > 0 {
			if config.Force {
				err := os.RemoveAll(dir)
				if err != nil {
					log.Fatalf("Unable to force clean dir `%s`: %v", dir, err)
				}
			} else {
				log.Fatalf("Pull target `%s` is not an empty directory, add --force to override", dir)
			}
		}
		dirFD.Close()
		return true
	} else {
		if !gitInfo.IsDir() {
			log.Fatalf("Pull target `%s` is not a Git repo", dir)
		}
	}
	return false

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

func upToDate(err error) bool {
	return err == git.NoErrAlreadyUpToDate || strings.Contains(err.Error(), "already up-to-date")
}

func componentExistInList(componentName string, components []string) bool {
	for _, comp := range components {
		if componentName == comp {
			return true
		}
	}
	return false
}

func dirExistInList(dir string, repos []LocalGitRepo) bool {
	for _, repo := range repos {
		if dir == repo.AbsDir {
			return true
		}
	}
	return false
}
