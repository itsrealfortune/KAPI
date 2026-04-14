package gitconfig

type GitConfig struct {
	InitLocal      bool
	HasExistingGit bool

	UniversalGitignore bool
	InitialCommit      bool

	RemoteURL     string
	RemoteHost    string
	RemotePrivate bool
	RepoName      string
	Collab        bool
	CI            string
}
