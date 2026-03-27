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

	// Could be empty for non-CMake or non-Meson projects.
	buildDir string

	repoDir string
}

func NewBuildProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) *BuildProject {
	return &BuildProject{
		Repo:     repo,
		BuildEnv: buildEnv,
		LibType:  libType,
		Shell:    buildEnv.Shell,
		OS:       buildEnv.OSEnv,
		CLIArgs:  buildEnv.Shell.Args,
	}
}

func (bp *BuildProject) GetKuBuiltinEnv() []string {
	b := bp.BuildEnv
	e := b.OSEnv
	env := []string{
		"KU_SDK=" + string(e.SDK),
		"KU_ARCH=" + string(e.Arch),
		"KU_ARCH_DIR=" + b.ArchDir,
		"KU_TARGET=" + b.Target,
		"KU_TARGET_LIB_NAME=" + b.TargetLibName,
		"KU_TARGET_DIR=" + b.TargetDir,
		"KU_OUT_DIR=" + b.OutDir,
		"KU_OUT_INCLUDE_DIR=" + b.OutIncludeDir,
		"KU_OUT_LIB_DIR=" + b.OutLibDir,
	}
	if b.DistDir != "" {
		env = append(env,
			"KU_DIST_DIR="+b.DistDir,
			"KU_DIST_INCLUDE_DIR="+b.DistIncludeDir,
			"KU_DIST_LIB_DIR="+b.DistLibDir,
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
