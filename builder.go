package ku

import (
	"fmt"
	"path/filepath"

	"github.com/mgenware/ku-builder/io2"
)

type Builder struct {
	Repo     *RepoInfo
	BuildEnv *BuildEnv

	LibType LibType

	// For convenience.
	Shell   *Shell
	OS      *OSEnv
	CLIArgs *CLIArgs

	repoRootDir string
	// Could be empty for non-CMake or non-Meson projects.
	buildDir string
}

func NewBuilder(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) *Builder {
	return &Builder{
		Repo:        repo,
		BuildEnv:    buildEnv,
		LibType:     libType,
		Shell:       buildEnv.Shell,
		OS:          buildEnv.OSEnv,
		CLIArgs:     buildEnv.Shell.Args,
		repoRootDir: generateRepoRootDir(repo),
	}
}

// `setup` indicates if it is called in the setup phase (e.g. Cmake generate or Meson setup).
func (bp *Builder) GetKuBuiltinEnv(setup bool) []string {
	be := bp.BuildEnv
	e := be.OSEnv

	libTypeExt := e.LibTypeExt(bp.LibType)
	targetLibFileName := be.TargetLibName + libTypeExt
	pkgConfigLibDir := filepath.Join(be.OutDir, "lib", "pkgconfig")

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

	if setup {
		env = append(env,
			"PKG_CONFIG="+bp.OS.GetPkgConfigPath(),
			// Force pkg-config to only look in our output directory for .pc files. This is needed to prevent pkg-config from auto-detecting libraries from the host system.
			"PKG_CONFIG_LIBDIR="+pkgConfigLibDir,
		)
	}

	return env
}

func (bp *Builder) NotNullOrQuit(v interface{}, name string) {
	if v == nil {
		bp.BuildEnv.Shell.Quit(fmt.Sprintf("%s should not be nil", name))
	}
}

func (bp *Builder) createBuildDir(repoName string, cleanBuild bool) {
	buildEnv := bp.BuildEnv
	buildDir := filepath.Join(buildEnv.TmpDir, repoName)
	if buildEnv.Shell.Args.CleanBuild || cleanBuild {
		io2.CleanDir(buildDir)
	} else {
		io2.Mkdirp(buildDir)
	}
	bp.buildDir = buildDir
}

func (bp *Builder) mustGetBuildDir(cleanBuild bool) string {
	if bp.buildDir == "" {
		bp.createBuildDir(bp.Repo.Name, cleanBuild)
	}
	return bp.buildDir
}

// Could be empty for non-CMake or non-Meson projects.
func (bp *Builder) GetBuildDir() string {
	return bp.buildDir
}

func generateRepoRootDir(repo *RepoInfo) string {
	if repo.LocalRepoDir != "" {
		return repo.LocalRepoDir
	}
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
