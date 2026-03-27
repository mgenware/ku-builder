package ku

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type BuildContext struct {
	Shell   *Shell
	Env     *Env
	CLIArgs *CLIArgs

	SDK  SDKEnum
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

	DebugBuild bool
	CleanBuild bool
}

type BuildContextInitOptions struct {
	Tunnel  *j9.Tunnel
	SDK     SDKEnum
	Arch    ArchEnum
	CLIArgs *CLIArgs
}

func NewBuildContextInitOpt(tunnel *j9.Tunnel, sdk SDKEnum, arch ArchEnum, cliArgs *CLIArgs) *BuildContextInitOptions {
	return &BuildContextInitOptions{
		Tunnel:  tunnel,
		SDK:     sdk,
		Arch:    arch,
		CLIArgs: cliArgs,
	}
}

func NewBuildContext(opt *BuildContextInitOptions) *BuildContext {
	if opt == nil {
		panic("opt is nil")
	}
	cliArgs := opt.CLIArgs
	buildTypeDir := GetBuildDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildTypeDir, opt.SDK)
	archDir := filepath.Join(sdkDir, string(opt.Arch))
	target := cliArgs.Target
	targetDir := filepath.Join(archDir, target)
	outDir := filepath.Join(targetDir, OutDirName)
	tmpDir := filepath.Join(targetDir, "tmp")
	shell := NewShell(opt.Tunnel)

	// Validate arch.
	sdkArchs := SDKArchs[opt.SDK]
	if !slices.Contains(sdkArchs, opt.Arch) {
		shell.Quit(fmt.Sprintf("Unsupported arch %s for SDK %s, valid archs: %v", opt.Arch, opt.SDK, sdkArchs))
	}
	outIncludeDir := filepath.Join(outDir, "include")
	outLibDir := filepath.Join(outDir, "lib")

	io2.Mkdirp(outIncludeDir)
	io2.Mkdirp(outLibDir)

	env := NewEnv(shell, cliArgs, opt.SDK, opt.Arch)

	ctx := &BuildContext{
		Shell:   shell,
		CLIArgs: opt.CLIArgs,
		Env:     env,

		SDK:    opt.SDK,
		Arch:   opt.Arch,
		Target: target,

		BuildTypeDir:  buildTypeDir,
		SDKDir:        sdkDir,
		ArchDir:       archDir,
		TmpDir:        tmpDir,
		TargetDir:     targetDir,
		OutDir:        outDir,
		OutIncludeDir: outIncludeDir,
		OutLibDir:     outLibDir,
		DebugBuild:    cliArgs.DebugBuild,
		CleanBuild:    cliArgs.CleanBuild,
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

func (ctx *BuildContext) LogContext() {
	ctx.Shell.Logger().Log(j9.LogLevelWarning, "Building target: "+ctx.CLIArgs.Target+"-"+string(ctx.Arch)+"-"+string(ctx.SDK))
}

func (ctx *BuildContext) VerifyOutLibFileArch(outFile []string) {
	baseDir := ctx.OutLibDir
	ctx.Env.AutoVerifyFileArch(baseDir, outFile)
}

func (ctx *BuildContext) VerifyDistLibFileArch(outFile []string) {
	baseDir := ctx.DistLibDir
	ctx.Env.AutoVerifyFileArch(baseDir, outFile)
}

func (ctx *BuildContext) GetArchBuildDir(repoName string) string {
	buildDir := filepath.Join(ctx.TmpDir, repoName)
	if ctx.CleanBuild {
		io2.CleanDir(buildDir)
	} else {
		io2.Mkdirp(buildDir)
	}
	return buildDir
}

func (ctx *BuildContext) GetCoreKuEnv() []string {
	env := []string{
		"KU_SDK=" + string(ctx.SDK),
		"KU_ARCH=" + string(ctx.Arch),
		"KU_ARCH_DIR=" + ctx.ArchDir,
		"KU_TARGET=" + ctx.Target,
		"KU_TARGET_LIB_NAME=" + ctx.TargetLibName,
		"KU_TARGET_DIR=" + ctx.TargetDir,
		"KU_OUT_DIR=" + ctx.OutDir,
		"KU_OUT_INCLUDE_DIR=" + ctx.OutIncludeDir,
		"KU_OUT_LIB_DIR=" + ctx.OutLibDir,
	}
	if ctx.DistDir != "" {
		env = append(env,
			"KU_DIST_DIR="+ctx.DistDir,
			"KU_DIST_INCLUDE_DIR="+ctx.DistIncludeDir,
			"KU_DIST_LIB_DIR="+ctx.DistLibDir,
		)
	}
	return env
}

func (ctx *BuildContext) NotNullOrQuit(v interface{}, name string) {
	if v == nil {
		ctx.Shell.Quit(fmt.Sprintf("%s should not be nil", name))
	}
}
