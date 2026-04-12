package ku

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type BuildEnv struct {
	Shell *Shell
	OSEnv *OSEnv

	// Returns Shell.Args.
	CLIArgs *CLIArgs
	// Returns OSEnv.SDK.
	SDK SDKEnum
	// Returns OSEnv.Arch.
	Arch ArchEnum

	// Example: ffmpeg
	Target string
	// Example: libffmpeg
	TargetLibName string

	// BuildTypeDir = ${RootBuildDir}/${release/debug}
	BuildTypeDir string
	// SDKDir = ${BuildTypeDir}/${Platform}/${SDK}
	SDKDir string
	// ArchDir = ${BuildTypeDir}/${Platform}/${SDK}/${Arch}
	ArchDir string

	// TargetDir: ${ArchDir}/${Target}
	TargetDir string

	// Out = ${TargetDir}/out
	OutDir string
	// ${OutDir}/include
	OutIncludeDir string
	// ${OutDir}/lib
	OutLibDir string

	// Optional dist dir for the target (some libraries might use it to output final products).
	DistDir string
	// ${DistDir}/include
	DistIncludeDir string
	// ${DistDir}/lib
	DistLibDir string

	// TmpDir = ${TargetDir}/tmp
	// Mostly CMake build files.
	TmpDir string
}

func NewBuildEnv(shell *Shell, env *OSEnv) *BuildEnv {
	cliArgs := shell.Args

	buildTypeDir := GetBuildTypeDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildTypeDir, env.SDK)
	archDir := filepath.Join(sdkDir, string(env.Arch))
	target := cliArgs.Target
	targetDir := filepath.Join(archDir, target)
	outDir := filepath.Join(targetDir, OutDirName)
	tmpDir := filepath.Join(targetDir, "tmp")

	// Validate arch.
	sdkArchs := SDKArchs[env.SDK]
	if !slices.Contains(sdkArchs, env.Arch) {
		shell.Quit(fmt.Sprintf("Unsupported arch %s for SDK %s, valid archs: %v", env.Arch, env.SDK, sdkArchs))
	}
	outIncludeDir := filepath.Join(outDir, "include")
	outLibDir := filepath.Join(outDir, "lib")

	io2.Mkdirp(outIncludeDir)
	io2.Mkdirp(outLibDir)

	ctx := &BuildEnv{
		Shell:   shell,
		OSEnv:   env,
		CLIArgs: shell.Args,
		SDK:     env.SDK,
		Arch:    env.Arch,

		Target: target,

		BuildTypeDir:  buildTypeDir,
		SDKDir:        sdkDir,
		ArchDir:       archDir,
		TmpDir:        tmpDir,
		TargetDir:     targetDir,
		OutDir:        outDir,
		OutIncludeDir: outIncludeDir,
		OutLibDir:     outLibDir,
	}

	targetLibName := GetTargetLibName(target)
	ctx.TargetLibName = targetLibName

	if cliArgs.Options != nil && cliArgs.Options.CreateDistDir {
		distDir := filepath.Join(targetDir, DistDirName)
		distIncludeDir := filepath.Join(distDir, "include")
		distLibDir := filepath.Join(distDir, "lib")
		io2.Mkdirp(distIncludeDir)
		io2.Mkdirp(distLibDir)

		ctx.DistDir = distDir
		ctx.DistIncludeDir = distIncludeDir
		ctx.DistLibDir = distLibDir
	}

	return ctx
}

func (be *BuildEnv) LogSummary() {
	shell := be.Shell
	cliArgs := shell.Args
	osEnv := be.OSEnv

	shell.Logger().Log(j9.LogLevelWarning, "Building target: "+cliArgs.Target+"-"+string(osEnv.SDK)+"-"+string(osEnv.Arch))
}

func (e *BuildEnv) VerifyLibFileArch(outFile []string) {
	e.OSEnv.AutoVerifyFileArch(e.OutLibDir, e.DistLibDir, outFile)
}
