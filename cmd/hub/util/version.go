package util

import "fmt"

var (
	ref     = "master"
	commit  = "HEAD"
	buildAt = "now"
)

func Version() string {
	return fmt.Sprintf("%s %s build at %s", ref, commit, buildAt)
}
