package ku

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

func getRepoDir(repo *RepoInfo) string {
	var ver string
	if repo.Tag != "" {
		ver = repo.Tag
	} else if repo.Commit != "" {
		ver = repo.Commit
	} else if repo.UrlArchiveName != "" {
		ver = repo.UrlArchiveName
	} else if repo.Branch != "" {
		ver = repo.Branch
	} else {
		ver = "_latest_"
	}
	return filepath.Join(GlobalRepoDir, string(repo.Name), ver)
}

func (bp *BuildProject) CloneAndGotoRepo() string {
	repo := bp.Repo
	shell := bp.Shell
	repoDir := getRepoDir(repo)

	if io2.DirectoryExists(repoDir) && !checkDirEmpty(repoDir) {
		shell.CD(repoDir)
		return repoDir
	}

	io2.Mkdirp(repoDir)
	shell.CD(repoDir)

	if repo.UrlArchiveName != "" {
		if !repo.CreateArchiveDirName {
			// If `CreateArchiveDirName` is false, we assume the archive contains a root directory named `ArchiveDirName`.
			// We go to the parent directory.
			shell.CD("..")
		}
		// Download the archive and extract it.
		tmpFile, err := os.CreateTemp("", "ku_download")
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpFile.Name())

		shell.Spawn(&j9.SpawnOpt{
			Name: "curl",
			Args: []string{"-L", "-o", tmpFile.Name(), repo.Url},
		})

		var tarFlags string
		if strings.HasSuffix(repo.Url, ".tar.gz") {
			tarFlags = "-xzvf"
		} else {
			tarFlags = "-xvf"
		}
		shell.Spawn(&j9.SpawnOpt{
			Name: "tar",
			Args: []string{tarFlags, tmpFile.Name()},
		})

		if !repo.CreateArchiveDirName {
			shell.CD(repoDir)
		}

		return repoDir
	}

	var args []string
	needCheckout := false
	if repo.Tag != "" {
		args = []string{"clone", "--branch", repo.Tag, "--depth", "1", repo.Url, repoDir}
	} else if repo.Commit != "" {
		args = []string{"clone", repo.Url, repoDir}
		needCheckout = true
	} else if repo.Branch != "" {
		args = []string{"clone", "--branch", repo.Branch, "--depth", "1", repo.Url, repoDir}
	} else {
		args = []string{"clone", "--depth", "1", repo.Url, repoDir}
	}

	shell.Spawn(&j9.SpawnOpt{
		Name: "git",
		Args: args,
	})

	if needCheckout {
		shell.Spawn(&j9.SpawnOpt{
			Name: "git",
			Args: []string{"-C", repoDir, "checkout", repo.Commit},
		})
	}

	if repo.PostCheckoutCommands != nil {
		for _, cmd := range repo.PostCheckoutCommands {
			shell.Spawn(&j9.SpawnOpt{
				Name: cmd[0],
				Args: cmd[1:],
			})
		}
	}

	bp.repoDir = repoDir
	return repoDir
}

func (bp *BuildProject) GetRepoDir() string {
	return bp.repoDir
}

func checkDirEmpty(path string) bool {
	empty, err := io2.IsDirectoryEmpty(path)
	if err != nil {
		panic(err)
	}
	return empty
}
