package ku

import (
	"fmt"

	"github.com/mgenware/j9/v3"
)

type BuildProject struct {
	Repo     *RepoInfo
	BuildEnv *BuildEnv

	LibType  LibType
	BuildDir string

	// For convenience.
	Shell   *Shell
	OS      *OSEnv
	CLIArgs *CLIArgs
}

func NewBuildProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) *BuildProject {
	buildDir := buildEnv.CreateBuildDir(repo.Name)
	return &BuildProject{
		Repo:     repo,
		BuildEnv: buildEnv,
		LibType:  libType,
		BuildDir: buildDir,
		Shell:    buildEnv.Shell,
		OS:       buildEnv.OSEnv,
		CLIArgs:  buildEnv.Shell.Args,
	}
}

func (bp *BuildProject) LogSummary() {
	buildEnv := bp.BuildEnv
	shell := buildEnv.Shell
	cliArgs := shell.Args
	osEnv := buildEnv.OSEnv

	shell.Logger().Log(j9.LogLevelWarning, "Building target: "+cliArgs.Target+"-"+string(osEnv.SDK)+"-"+string(osEnv.Arch)+"-"+bp.LibType.String())
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
