package ku

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type BuildContext struct {
	Shell   *Shell
	Env     *Env
	CLIArgs *CLIArgs
	LibType LibType

	SDK  SDKEnum
	Arch ArchEnum
	// Example: ffmpeg
	Target string
	// Example: libffmpeg
	TargetLibName string
	// Example: libffmpeg.dylib
	TargetLibFileName string

	// BuildDir = ${RootBuildDir}/${release/debug}
	BuildDir string
	// SDKDir = ${BuildDir}/${Platform}/${SDK}
	SDKDir string
	// ArchDir = ${BuildDir}/${Platform}/${SDK}/${Arch}
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

	stringCache map[string]string
}

type BuildContextInitOptions struct {
	Tunnel  *j9.Tunnel
	SDK     SDKEnum
	Arch    ArchEnum
	CLIArgs *CLIArgs
	LibType LibType
}

func NewBuildContextInitOpt(tunnel *j9.Tunnel, sdk SDKEnum, arch ArchEnum, cliArgs *CLIArgs, libType LibType) *BuildContextInitOptions {
	return &BuildContextInitOptions{
		Tunnel:  tunnel,
		SDK:     sdk,
		Arch:    arch,
		CLIArgs: cliArgs,
		LibType: libType,
	}
}

func NewBuildContext(opt *BuildContextInitOptions) *BuildContext {
	if opt == nil {
		panic("opt is nil")
	}
	cliArgs := opt.CLIArgs
	buildDir := GetBuildDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildDir, opt.SDK)
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

		SDK:     opt.SDK,
		Arch:    opt.Arch,
		LibType: opt.LibType,
		Target:  target,

		BuildDir:      buildDir,
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
	targetLibFileName := targetLibName + ctx.Env.GetLibExt(ctx.LibType)

	ctx.TargetLibName = targetLibName
	ctx.TargetLibFileName = targetLibFileName

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

func (ctx *BuildContext) RunMakeInstall(outFile []string) {
	env := ctx.GetCoreKuEnv()
	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"install"},
		Env:  env,
	})
	ctx.VerifyOutFileArch(outFile)
}

func (ctx *BuildContext) VerifyOutFileArch(outFile []string) {
	if len(outFile) > 0 {
		parts := append([]string{ctx.OutLibDir}, outFile...)
		outPath := filepath.Join(parts...)
		outPath = outPath + ctx.Env.GetLibExt(ctx.LibType)
		ctx.Env.VerifyFileArch(ctx.LibType, outPath)
	}
}

func (ctx *BuildContext) RunMakeCleanRaw() error {
	env := ctx.GetCoreKuEnv()
	return ctx.Shell.SpawnRaw(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"clean"},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMakeClean() {
	err := ctx.RunMakeCleanRaw()
	if err != nil {
		panic(err)
	}
}

func (ctx *BuildContext) RunMakeWithArgs(opt *j9.SpawnOpt) {
	if opt == nil {
		opt = &j9.SpawnOpt{}
	}
	numCores := runtime.NumCPU()

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)

	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{fmt.Sprintf("-j%v", numCores)},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMake() {
	ctx.RunMakeWithArgs(nil)
}

type RunCmakeGenOptions struct {
	Args   []string
	Env    []string
	Preset string
}

func (ctx *BuildContext) RunCmakeGen(opt *RunCmakeGenOptions) {
	args := opt.Args
	if ctx.CleanBuild {
		args = append(args, "--fresh")
	}
	if opt.Preset != "" {
		args = append(args, "--preset", opt.Preset)
	}

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)
	env = append(env,
		"KU_CMAKE_ACTION=gen",
	)

	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: args,
		Env:  env,
	})
}

type CmakeActionType string

const (
	CmakeActionBuild   CmakeActionType = "build"
	CmakeActionInstall CmakeActionType = "install"
)

type RunCmakeBuildOrInstallOptions struct {
	// Required.
	Action CmakeActionType

	Target    string
	ExtraArgs []string
	Env       []string
}

func (ctx *BuildContext) RunCmakeBuildOrInstall(opt *RunCmakeBuildOrInstallOptions, outFile []string) {
	if opt == nil {
		panic("opt is nil")
	}
	if opt.Action == "" {
		panic("opt.Action is empty")
	}

	args := []string{
		"--" + string(opt.Action), ".",
	}

	if opt.Target != "" {
		if opt.Action == CmakeActionInstall {
			panic("opt.Target is not supported for install")
		}
		args = append(args, "--target", opt.Target)
	}

	var config string
	if ctx.DebugBuild {
		config = "Debug"
	} else {
		config = "Release"
	}
	args = append(args, "--config", config)

	// Strip during production install.
	if opt.Action == CmakeActionInstall && !ctx.DebugBuild {
		// This uses `CMAKE_STRIP`, which is set by Android toolchain.
		args = append(args, "--strip")
	}

	numCores := runtime.NumCPU()
	args = append(args, "-j", fmt.Sprintf("%v", numCores))

	// Extra args.
	if len(opt.ExtraArgs) > 0 {
		args = append(args, opt.ExtraArgs...)
	}

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)
	env = append(env,
		"KU_CMAKE_ACTION="+string(opt.Action),
	)
	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: args,
		Env:  env,
	})

	ctx.VerifyOutFileArch(outFile)
}

func (ctx *BuildContext) RunCmakeBuild() {
	ctx.RunCmakeBuildTarget("")
}

func (ctx *BuildContext) RunCmakeBuildTarget(target string) {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: CmakeActionBuild,
		Target: target,
	}
	ctx.RunCmakeBuildOrInstall(opt, nil)
}

func (ctx *BuildContext) RunCmakeInstall(outFile []string) {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: CmakeActionInstall,
	}
	ctx.RunCmakeBuildOrInstall(opt, outFile)
}

func (ctx *BuildContext) LogContext() {
	ctx.Shell.Logger().Log(j9.LogLevelWarning, "Building target: "+ctx.CLIArgs.Target+"-"+string(ctx.Arch)+"-"+string(ctx.SDK)+"-"+ctx.LibType.String())
}

type GetCompilerFlagsOptions struct {
	LD          bool
	DisableArch bool
	EnablePIC   bool
}

func (ctx *BuildContext) getCompilerFlagsList(opt *GetCompilerFlagsOptions) []string {
	if opt == nil {
		opt = &GetCompilerFlagsOptions{}
	}
	args := []string{}

	if ctx.Env.IsDarwinPlatform() {
		archStr := string(ctx.Arch)
		if !opt.DisableArch {
			args = append(args, "-arch", archStr)
		}

		args = append(args, "-isysroot", ctx.Env.GetSDKRootPath())

		// Darwin -target and min SDK version.
		switch ctx.SDK {
		case SDKMacos:
			args = append(args, "-target", archStr+"-apple-macosx"+MinMacosVersion)
			args = append(args, "-mmacosx-version-min="+MinMacosVersion)
		case SDKIosSimulator:
			args = append(args, "-target", archStr+"-apple-ios"+MinIosVersion+"-simulator")
			args = append(args, "-mios-simulator-version-min="+MinIosVersion)
		case SDKIos:
			args = append(args, "-target", archStr+"-apple-ios"+MinIosVersion)
			args = append(args, "-miphoneos-version-min="+MinIosVersion)
		}
	}

	if ctx.DebugBuild {
		args = append(args, "-g")
	}

	if opt.EnablePIC {
		args = append(args, "-fPIC")
	}

	return args
}

func (ctx *BuildContext) GetCompilerFlags(opt *GetCompilerFlagsOptions) string {
	return strings.Join(ctx.getCompilerFlagsList(opt), " ")
}

type GetCompilerConfigureEnvOptions struct {
	// When true, override CFLAGS, CXXFLAGS, LDFLAGS.
	// Useful for make projects using `./configure`.
	// Note that might override existing compiler flags provided by source repo.
	// In that case, it's recommended to use `--extra-xxxflags` during `./configure`.
	OverrideCompilerFlags bool
}

// GetCompilerConfigureEnv returns environment variables for compiler configuration.
// This includes CC, CXX, LD, and optionally CFLAGS, CXXFLAGS, LDFLAGS (when OverrideCompilerFlags is true).
// On Android, it also includes AR, AS, RANLIB, STRIP, NM.
func (ctx *BuildContext) GetCompilerConfigureEnv(opt *GetCompilerConfigureEnvOptions) []string {
	if opt == nil {
		opt = &GetCompilerConfigureEnvOptions{}
	}

	args := []string{
		"CC=" + ctx.Env.GetCCPath(),
		"CXX=" + ctx.Env.GetCXXPath(),
		"LD=" + ctx.Env.GetLDPath(),
	}
	if ctx.Env.IsAndroidPlatform() {
		args = append(args, "AR="+ctx.Env.GetNDKToolchainBinPath("llvm-ar"))
		args = append(args, "AS="+ctx.Env.GetNDKToolchainBinPath("llvm-as"))
		args = append(args, "RANLIB="+ctx.Env.GetNDKToolchainBinPath("llvm-ranlib"))
		args = append(args, "STRIP="+ctx.Env.GetNDKToolchainBinPath("llvm-strip"))
		args = append(args, "NM="+ctx.Env.GetNDKToolchainBinPath("llvm-nm"))
	}

	if opt.OverrideCompilerFlags {
		cflags := ctx.GetCompilerFlags(nil)
		ldflags := ctx.GetCompilerFlags(&GetCompilerFlagsOptions{LD: true})

		args = append(args, "CFLAGS="+cflags)
		args = append(args, "CXXFLAGS="+cflags)
		args = append(args, "LDFLAGS="+ldflags)
	}

	return args
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

type CommonCmakeArgsOptions struct {
	EnableSystemPath bool
	DisablePIC       bool
}

func (ctx *BuildContext) CommonCmakeArgs() []string {
	return ctx.CommonCmakeArgsWithOptions(nil)
}

func (ctx *BuildContext) CommonCmakeArgsWithOptions(opt *CommonCmakeArgsOptions) []string {
	if opt == nil {
		opt = &CommonCmakeArgsOptions{}
	}

	libType := ctx.LibType
	var isDylib bool
	if SupportedLibTypes[libType] {
		isDylib = libType == LibTypeDynamic
	} else {
		ctx.Shell.Quit(fmt.Sprintf("Invalid libType: %s, valid types: %v", libType, SupportedLibTypes))
	}

	var targetOS string
	switch ctx.SDK {
	case SDKMacos:
		targetOS = "Darwin"
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		targetOS = "iOS"
	case SDKAndroid:
		targetOS = "Android"
	}
	ctx.Shell.Logger().Log(j9.LogLevelVerbose, "[Cmake] Target OS: "+targetOS)

	args := []string{
		"-DCMAKE_SYSTEM_NAME=" + targetOS,
		"-DCMAKE_INSTALL_PREFIX=" + ctx.OutDir,
		"-DCMAKE_PREFIX_PATH=" + ctx.OutDir,
		"-DCMAKE_LIBRARY_PATH=" + ctx.OutLibDir,
	}

	if !opt.EnableSystemPath {
		args = append(args,
			"-DCMAKE_FIND_USE_CMAKE_SYSTEM_PATH=0",
			"-DCMAKE_FIND_USE_SYSTEM_ENVIRONMENT_PATH=0",
		)
	}

	if !opt.DisablePIC {
		args = append(args,
			"-DCMAKE_POSITION_INDEPENDENT_CODE=1",
		)
	}

	isDylibStr := "0"
	if isDylib {
		isDylibStr = "1"
	}
	args = append(args, "-DBUILD_SHARED_LIBS="+isDylibStr)

	if ctx.Env.IsDarwinPlatform() {
		args = append(args,
			// SDK
			"-DCMAKE_OSX_SYSROOT="+ctx.Env.GetSDKRootPath(),
			// Min SDK
			"-DCMAKE_OSX_DEPLOYMENT_TARGET="+ctx.Env.MinDarwinSDKVer(),
			// -arch
			"-DCMAKE_OSX_ARCHITECTURES="+string(ctx.Arch),
			"-DCMAKE_MACOSX_BUNDLE=0",
			"-DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_ALLOWED=0",
			// On Android, this should be set by `DCMAKE_TOOLCHAIN_FILE`.
			"-DCMAKE_SYSTEM_PROCESSOR="+string(ctx.Arch),
		)
	}

	if ctx.Env.IsAndroidPlatform() {
		ndk := ctx.Env.GetNDKPath()
		abi := GetABI(ctx.Arch)
		args = append(args,
			"-DANDROID_NDK="+ndk,
			"-DANDROID_ABI="+abi,
			"-DANDROID_PLATFORM=android-"+MinAndroidAPI,
			"-DCMAKE_ANDROID_NDK="+ndk,
			"-DCMAKE_TOOLCHAIN_FILE="+ctx.Env.GetNDKCmakeToolchainFile(),
			"-DCMAKE_ANDROID_ARCH_ABI="+abi,
			"-DCMAKE_SYSTEM_VERSION="+MinAndroidAPI,
		)
	}

	var buildType string
	if ctx.DebugBuild {
		buildType = "Debug"
	} else {
		buildType = "Release"
	}
	args = append(args, "-DCMAKE_BUILD_TYPE="+buildType)
	return args
}

func (ctx *BuildContext) GetCoreKuEnv() []string {
	env := []string{
		"KU_SDK=" + string(ctx.SDK),
		"KU_ARCH=" + string(ctx.Arch),
		"KU_ARCH_DIR=" + ctx.ArchDir,
		"KU_TARGET=" + ctx.Target,
		"KU_TARGET_LIB_NAME=" + ctx.TargetLibName,
		"KU_TARGET_LIB_FILENAME=" + ctx.TargetLibFileName,
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
