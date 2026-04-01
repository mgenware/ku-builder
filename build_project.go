package ku

import (
	"fmt"
	"path/filepath"

	"github.com/mgenware/ku-builder/io2"
)

type BuildProject struct {
	Repo     *RepoInfo
	BuildEnv *BuildEnv

	LibType LibType

	// For convenience.
	Shell   *Shell
	OS      *OSEnv
	CLIArgs *CLIArgs

	repoDir string
	// Could be empty for non-CMake or non-Meson projects.
	buildDir string
}

func NewBuildProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) *BuildProject {
	return &BuildProject{
		Repo:     repo,
		BuildEnv: buildEnv,
		LibType:  libType,
		Shell:    buildEnv.Shell,
		OS:       buildEnv.OSEnv,
		CLIArgs:  buildEnv.Shell.Args,
		repoDir:  generateRepoDir(repo),
	}
}

func (bp *BuildProject) GetKuBuiltinEnv() []string {
	be := bp.BuildEnv
	e := be.OSEnv

	libTypeExt := e.LibTypeExt(bp.LibType)
	targetLibFileName := be.TargetLibName + libTypeExt
	env := []string{
		"KU_SDK=" + string(e.SDK),
		"KU_ARCH=" + string(e.Arch),
		"KU_ARCH_DIR=" + be.ArchDir,
		"KU_TARGET=" + be.Target,
		"KU_TARGET_LIB_NAME=" + be.TargetLibName,
		"KU_TARGET_DIR=" + be.TargetDir,
		"KU_OUT_DIR=" + be.OutDir,
		"KU_OUT_INCLUDE_DIR=" + be.OutIncludeDir,
		"KU_OUT_LIB_DIR=" + be.OutLibDir,
		"KU_LIB_TYPE=" + bp.LibType.String(),
		"KU_LIB_TYPE_EXT=" + libTypeExt,
		"KU_TARGET_LIB_FILENAME=" + targetLibFileName,
	}
	if be.DistDir != "" {
		env = append(env,
			"KU_DIST_DIR="+be.DistDir,
			"KU_DIST_INCLUDE_DIR="+be.DistIncludeDir,
			"KU_DIST_LIB_DIR="+be.DistLibDir,
		)
	}
	return env
}

func (bp *BuildProject) NotNullOrQuit(v interface{}, name string) {
	if v == nil {
		bp.BuildEnv.Shell.Quit(fmt.Sprintf("%s should not be nil", name))
	}
}

func (bp *BuildProject) createBuildDir(repoName string) {
	buildEnv := bp.BuildEnv
	buildDir := filepath.Join(buildEnv.TmpDir, repoName)
	if buildEnv.Shell.Args.CleanBuild {
		io2.CleanDir(buildDir)
	} else {
		io2.Mkdirp(buildDir)
	}
	bp.buildDir = buildDir
}

func (bp *BuildProject) mustGetBuildDir() string {
	if bp.buildDir == "" {
		bp.createBuildDir(bp.Repo.Name)
	}
	return bp.buildDir
}

// Could be empty for non-CMake or non-Meson projects.
func (bp *BuildProject) GetBuildDir() string {
	return bp.buildDir
}

func generateRepoDir(repo *RepoInfo) string {
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
