package ku

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

func GetRepoDir(repo *SourceRepo) string {
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
	return filepath.Join(ProjectRepoDir, string(repo.Name), ver)
}

func CloneAndGotoRepo(t *j9.Tunnel, repo *SourceRepo) string {
	dir := GetRepoDir(repo)

	if io2.DirectoryExists(dir) && !checkDirEmpty(dir) {
		t.CD(dir)
		return dir
	}

	io2.Mkdirp(dir)
	t.CD(dir)

	if repo.UrlArchiveName != "" {
		if !repo.CreateArchiveDirName {
			// If `CreateArchiveDirName` is false, we assume the archive contains a root directory named `ArchiveDirName`.
			// We go to the parent directory.
			t.CD("..")
		}
		// Download the archive and extract it.
		tmpFile, err := os.CreateTemp("", "ku_download")
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpFile.Name())

		t.Spawn(&j9.SpawnOpt{
			Name: "curl",
			Args: []string{"-L", "-o", tmpFile.Name(), repo.Url},
		})

		var tarFlags string
		if strings.HasSuffix(repo.Url, ".tar.gz") {
			tarFlags = "-xzvf"
		} else {
			tarFlags = "-xvf"
		}
		t.Spawn(&j9.SpawnOpt{
			Name: "tar",
			Args: []string{tarFlags, tmpFile.Name()},
		})

		if !repo.CreateArchiveDirName {
			t.CD(dir)
		}

		return dir
	}

	var args []string
	needCheckout := false
	if repo.Tag != "" {
		args = []string{"clone", "--branch", repo.Tag, "--depth", "1", repo.Url, dir}
	} else if repo.Commit != "" {
		args = []string{"clone", repo.Url, dir}
		needCheckout = true
	} else if repo.Branch != "" {
		args = []string{"clone", "--branch", repo.Branch, "--depth", "1", repo.Url, dir}
	} else {
		args = []string{"clone", "--depth", "1", repo.Url, dir}
	}

	t.Spawn(&j9.SpawnOpt{
		Name: "git",
		Args: args,
	})

	if needCheckout {
		t.Spawn(&j9.SpawnOpt{
			Name: "git",
			Args: []string{"-C", dir, "checkout", repo.Commit},
		})
	}

	if repo.PostCheckoutCommands != nil {
		for _, cmd := range repo.PostCheckoutCommands {
			t.Spawn(&j9.SpawnOpt{
				Name: cmd[0],
				Args: cmd[1:],
			})
		}
	}

	return dir
}

func checkDirEmpty(path string) bool {
	empty, err := io2.IsDirectoryEmpty(path)
	if err != nil {
		panic(err)
	}
	return empty
}
