package git

type LocalGitRepo struct {
	Remote          string
	OptimizedRemote string
	Ref             string
	HeadRef         string
	SubDir          string
	AbsDir          string
}
