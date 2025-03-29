package ku

type SourceRepo struct {
	Url                  string
	Name                 string
	FolderName           string
	Tag                  string
	Commit               string
	UrlArchiveName       string
	CreateArchiveDirName bool
	PostCheckoutCommands [][]string
}

type SourceInfo struct {
	Repo *SourceRepo

	RepoDir string
	// Some repos like libaom require a separate build directory.
	BuildDir string
}

func NewSourceInfo(repo *SourceRepo, repoDir string) *SourceInfo {
	return &SourceInfo{
		Repo:    repo,
		RepoDir: repoDir,
	}
}
