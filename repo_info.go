package ku

type RepoInfo struct {
	Url  string
	Name string

	Tag            string
	Commit         string
	UrlArchiveName string
	Branch         string

	CreateArchiveDirName bool
	PostCheckoutCommands [][]string
}
