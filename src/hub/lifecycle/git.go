package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"

	"hub/util"
)

func gitStatus(dir string, calculateStatus bool) (map[string]string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("Unable to calculate absolute path of `%s`: %v", dir, err)
	}
	dir = abs
	for {
		gitDir := filepath.Join(dir, ".git")
		_, err = os.Stat(gitDir)
		if err != nil {
			if util.NoSuchFile(err) {
				parent := filepath.Dir(dir)
				if dir == parent {
					return map[string]string{
						"ref":   "(not a Git)",
						"clean": "",
					}, nil
				}
				dir = parent
				continue
			}
			return nil, fmt.Errorf("Unable to stat `%s`: %v", gitDir, err)
		}
		break
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("Unable to open Git repo in `%s`: %v", dir, err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("Unable to open Git worktree in `%s`: %v", dir, err)
	}
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("Unable to obtain Git repo HEAD info in `%s`: %v", dir, err)
	}
	clean := "not calculated"
	if calculateStatus {
		status, err := worktree.Status()
		if err != nil {
			return nil, fmt.Errorf("Unable to get Git status in `%s`: %v", dir, err)
		}
		if status.IsClean() {
			clean = "clean"
		} else {
			clean = "dirty"
		}
	}
	refs := head.Strings()
	name := refs[0]
	ref := refs[1]
	if len(ref) == 40 {
		ref = ref[:7]
	}
	return map[string]string{
		"ref":   fmt.Sprintf("%s %s", name, ref),
		"clean": clean,
	}, nil
}
