package ku

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type RepoInfo struct {
	// URL of the git repo.
	Url string
	// Name of the repo, used to create the repo directory.
	Name string
	// If set, use the local directory instead of cloning. The directory should already exist.
	LocalRepoDir string

	// If set, clone the repo and checkout the tag.
	Tag string
	// If set, clone the repo and checkout the commit.
	Commit string
	// If set, clone the repo and checkout the branch.
	Branch string

	// The archive file name (without extension) of the URL. If set, download the archive and extract it. The URL should point to an archive file (e.g. .tar.gz, .zip).
	UrlArchiveName string

	// If set, run these commands after checking out the repo.
	PostCheckoutCommands [][]string

	// If set, go to this subdirectory after setting up the repo. The path is relative to the repo root.
	// Some repos have the source code in a subdirectory instead of the repo root.
	SourceSubDir []string
}

var repoPulled = make(map[string]bool)

// Clones the repo if needed and goes to the repo directory. Returns the repo source directory.
func (bp *Builder) CloneAndGotoRepoSource() string {
	repoRootDir := bp.cloneAndGotoRepoRoot()

	srcDir := repoRootDir
	hasSubDir := len(bp.Repo.SourceSubDir) > 0
	if hasSubDir {
		srcDir = filepath.Join(repoRootDir, filepath.Join(bp.Repo.SourceSubDir...))

		if !io2.DirectoryExists(srcDir) {
			bp.Shell.Quit(fmt.Sprintf("Source subdirectory %s does not exist\n", srcDir))
		}
		bp.Shell.CD(srcDir)
	}
	return srcDir
}

func (bp *Builder) cloneAndGotoRepoRoot() string {
	repo := bp.Repo
	shell := bp.Shell
	repoDir := bp.repoRootDir

	if io2.DirectoryExists(repoDir) && !checkDirEmpty(shell, repoDir) {
		shell.CD(repoDir)

		// Call git pull if needed.
		if repo.LocalRepoDir == "" && repo.UrlArchiveName == "" && repo.Commit == "" && !bp.CLIArgs.NoPull {
			if !repoPulled[repoDir] {
				shell.Spawn(&j9.SpawnOpt{
					Name: "git",
					Args: []string{"pull"},
				})
				repoPulled[repoDir] = true
			}
		}

		return repoDir
	}

	io2.Mkdirp(repoDir)
	shell.CD(repoDir)

	if repo.UrlArchiveName != "" {
		// If `UrlArchiveName` is set, `repoDir` is now '<repo>/<UrlArchiveName>'. We need to go back to the parent directory to download and
		// extract the archive at `<repo>/`, which will create the `<repo>/<UrlArchiveName>/` directory.
		shell.CD("..")
		// Download the archive and extract it.
		tmpFile, err := os.CreateTemp("", "ku_download")
		if err != nil {
			shell.Quit(fmt.Sprintf("Error creating temp file: %v\n", err))
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

		shell.CD(repoDir)
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

	return repoDir
}

func (bp *Builder) GetRepoRootDir() string {
	return bp.repoRootDir
}

func checkDirEmpty(shell *Shell, path string) bool {
	empty, err := io2.IsDirectoryEmpty(path)
	if err != nil {
		shell.Quit(fmt.Sprintf("Error checking if directory is empty: %v\n", err))
	}
	return empty
}
