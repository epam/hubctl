package git

import (
	"fmt"
	"log"
)

func printLocalGitRepos(repos []LocalGitRepo) {
	for _, repo := range repos {
		remote := fmt.Sprintf("`%s`", repo.Remote)
		if repo.OptimizedRemote != repo.Remote {
			remote = fmt.Sprintf("`%s` (%s)", repo.OptimizedRemote, repo.Remote)
		}
		ref := fmt.Sprintf("`%s`", repo.HeadRef)
		if repo.Ref != "" && repo.Ref != repo.HeadRef {
			ref = fmt.Sprintf("`%s` (%s)", repo.HeadRef, repo.Ref)
		}
		subDir := ""
		if repo.SubDir != "" {
			subDir = fmt.Sprintf(" [/%s]", repo.SubDir)
		}
		log.Printf("\t%s => %s at %s%s", repo.AbsDir, remote, ref, subDir)
	}
}
