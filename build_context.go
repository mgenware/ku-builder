package ku

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type BuildContext struct {
	Tunnel  *j9.Tunnel
	CLIArgs *CLIArgs

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

	// Libs = ${TargetDir}/libs
	LibsDir string
	// ${LibsDir}/include
	LibsIncludeDir string
	// ${LibsDir}/lib
	LibsLibDir string

	// Optional out dir for the target.
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

	NDKInput string

	stringCache map[string]string
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
	buildDir := GetBuildDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildDir, opt.SDK)
	archDir := filepath.Join(sdkDir, string(opt.Arch))
	target := cliArgs.Target
	targetDir := filepath.Join(archDir, target)
	libsDir := filepath.Join(targetDir, "libs")
	tmpDir := filepath.Join(targetDir, "tmp")

	// Validate arch.
	sdkArchs := SDKArchs[opt.SDK]
	if !slices.Contains(sdkArchs, opt.Arch) {
		panic(fmt.Sprintf("Unsupported arch %s for SDK %s, valid archs: %v", opt.Arch, opt.SDK, sdkArchs))
	}
	libsIncludeDir := filepath.Join(libsDir, "include")
	libsLibDir := filepath.Join(libsDir, "lib")

	io2.Mkdirp(libsIncludeDir)
	io2.Mkdirp(libsLibDir)

	ctx := &BuildContext{
		Tunnel:  opt.Tunnel,
		CLIArgs: opt.CLIArgs,

		SDK:    opt.SDK,
		Arch:   opt.Arch,
		Target: target,

		BuildDir:       buildDir,
		SDKDir:         sdkDir,
		ArchDir:        archDir,
		TmpDir:         tmpDir,
		TargetDir:      targetDir,
		LibsDir:        libsDir,
		LibsIncludeDir: libsIncludeDir,
		LibsLibDir:     libsLibDir,
		DebugBuild:     cliArgs.DebugBuild,
		CleanBuild:     cliArgs.CleanBuild,
		NDKInput:       cliArgs.NDK,
	}

	targetLibName := GetTargetLibName(target)
	targetLibFileName := targetLibName + ctx.GetDylibExt()

	ctx.TargetLibName = targetLibName
	ctx.TargetLibFileName = targetLibFileName

	if cliArgs.Options != nil && cliArgs.Options.CreateDistDir {
		distDir := filepath.Join(targetDir, "dist")
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

func (ctx *BuildContext) RunMakeInstall() {
	env := ctx.GetCoreKuEnv()
	ctx.Tunnel.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"install"},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMakeClean() {
	env := ctx.GetCoreKuEnv()
	ctx.Tunnel.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"clean"},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMakeWithArgs(opt *j9.SpawnOpt) {
	if opt == nil {
		opt = &j9.SpawnOpt{}
	}
	numCores := runtime.NumCPU()

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)

	ctx.Tunnel.Spawn(&j9.SpawnOpt{
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

	ctx.Tunnel.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: args,
		Env:  env,
	})
}

type RunCmakeBuildOrInstallOptions struct {
	// Required.
	Action string

	Target    string
	ExtraArgs []string
	Env       []string
}

func (ctx *BuildContext) RunCmakeBuildOrInstall(opt *RunCmakeBuildOrInstallOptions) {
	if opt == nil {
		panic("opt is nil")
	}
	if opt.Action == "" {
		panic("opt.Action is empty")
	}

	args := []string{
		"--" + opt.Action, ".",
	}

	if opt.Target != "" {
		if opt.Action == "install" {
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
	if opt.Action == "install" && !ctx.DebugBuild {
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
		"KU_CMAKE_ACTION="+opt.Action,
	)
	ctx.Tunnel.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: args,
		Env:  env,
	})
}

func (ctx *BuildContext) RunCmakeBuild() {
	ctx.RunCmakeBuildTarget("")
}

func (ctx *BuildContext) RunCmakeBuildTarget(target string) {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: "build",
		Target: target,
	}
	ctx.RunCmakeBuildOrInstall(opt)
}

func (ctx *BuildContext) RunCmakeInstall() {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: "install",
	}
	ctx.RunCmakeBuildOrInstall(opt)
}

func (ctx *BuildContext) getSDKPathImpl() string {
	switch ctx.SDK {
	case SDKMacos:
		return ctx.ShellCmd("xcrun --sdk macosx --show-sdk-path")
	case SDKIos:
		return ctx.ShellCmd("xcrun --sdk iphoneos --show-sdk-path")
	case SDKIosSimulator:
		return ctx.ShellCmd("xcrun --sdk iphonesimulator --show-sdk-path")
	case SDKAndroid:
		return filepath.Join(ctx.getNDKToolchainRootPath(), "sysroot")
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetSDKPath() string {
	return ctx.cacheSDKArchString("sdk", func() string {
		return io2.DirectoryMustExist(ctx.getSDKPathImpl())
	})
}

type GetCompilerFlags struct {
	LD          bool
	DisableArch bool
	EnablePIC   bool
}

func (ctx *BuildContext) IsIosPlatform() bool {
	return ctx.SDK == SDKIos || ctx.SDK == SDKIosSimulator
}

func (ctx *BuildContext) IsDarwinPlatform() bool {
	return ctx.SDK == SDKMacos || ctx.IsIosPlatform()
}

func (ctx *BuildContext) IsAndroidPlatform() bool {
	return ctx.SDK == SDKAndroid
}

func (ctx *BuildContext) getCompilerFlagsList(opt *GetCompilerFlags) []string {
	if opt == nil {
		opt = &GetCompilerFlags{}
	}
	args := []string{}

	if ctx.IsDarwinPlatform() {
		if !opt.DisableArch {
			args = append(args, "-arch", string(ctx.Arch))
		}

		args = append(args, "-isysroot", ctx.GetSDKPath())

		// Darwin -target and min SDK version.
		switch ctx.SDK {
		case SDKMacos:
			// Min SDK.
			args = append(args, "-mmacosx-version-min="+MinMacosVersion)
		case SDKIosSimulator:
			args = append(args, "-mios-simulator-version-min="+MinIosVersion)
		case SDKIos:
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

func (ctx *BuildContext) GetCompilerFlags(opt *GetCompilerFlags) string {
	return strings.Join(ctx.getCompilerFlagsList(opt), " ")
}

func (ctx *BuildContext) getCCPathImpl() string {
	if ctx.IsDarwinPlatform() {
		return ctx.ShellCmd("xcodebuild -find clang")
	}
	if ctx.IsAndroidPlatform() {
		return ctx.getNDKClangPath(false)
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetCCPath() string {
	return ctx.cacheSDKArchString("cc", ctx.getCCPathImpl)
}

func (ctx *BuildContext) getCXXPathImpl() string {
	if ctx.IsDarwinPlatform() {
		return ctx.ShellCmd("xcodebuild -find clang++")
	}
	if ctx.IsAndroidPlatform() {
		return ctx.getNDKClangPath(true)
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetCXXPath() string {
	return ctx.cacheSDKArchString("cxx", ctx.getCXXPathImpl)
}

func (ctx *BuildContext) getLDPathImpl() string {
	if ctx.IsDarwinPlatform() {
		return ctx.GetCCPath()
	}
	if ctx.IsAndroidPlatform() {
		return ctx.getNDKClangPath(false)
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetLDPath() string {
	return ctx.cacheSDKArchString("ld", ctx.getLDPathImpl)
}

func (ctx *BuildContext) GetAndroidSDKPath() string {
	if ctx.IsAndroidPlatform() {
		return ctx.cacheSDKArchString("android_sdk", func() string {
			path := os.Getenv("ANDROID_SDK_PATH")
			if path != "" {
				return path
			}
			usr, err := os.UserHomeDir()
			if err != nil {
				panic(err)
			}
			path = filepath.Join(usr, "Library/Android/sdk")
			return path
		})
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetNDKPath() string {
	if ctx.IsAndroidPlatform() {
		return ctx.cacheSDKArchString("ndk", func() string {
			path := os.Getenv("ANDROID_NDK_PATH")
			if path != "" {
				return path
			}
			path = ctx.NDKInput
			// If `NDKInput` is not an absolute path, it's considered an NDK version.
			if !strings.HasPrefix(path, "/") {
				path = filepath.Join(ctx.GetAndroidSDKPath(), "ndk", path)
			}
			io2.DirectoryMustExist(path)
			return path
		})
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetNDKCmakeToolchainFile() string {
	if ctx.IsAndroidPlatform() {
		return ctx.cacheSDKArchString("ndk_cmake_toolchain", func() string {
			path := filepath.Join(ctx.GetNDKPath(), "build/cmake/android.toolchain.cmake")
			io2.FileMustExist(path)
			return path
		})
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) GetDylibExt() string {
	if ctx.IsDarwinPlatform() {
		return ".dylib"
	}
	return ".so"
}

func (ctx *BuildContext) getNDKToolchainRootPath() string {
	ndkPath := ctx.GetNDKPath()
	return ctx.cacheSDKArchString("ndk-toolchain-root", func() string {
		path := filepath.Join(ndkPath, "toolchains/llvm/prebuilt/darwin-x86_64")
		return io2.DirectoryMustExist(path)
	})
}

func (ctx *BuildContext) GetNDKToolchainBinPath(name string) string {
	return ctx.cacheSDKArchString("ndk-toolchain-bin-"+name, func() string {
		path := filepath.Join(ctx.getNDKToolchainRootPath(), "bin", name)
		return io2.FileMustExist(path)
	})
}

func (ctx *BuildContext) StripFile(src, dst string) {
	var stripBin string
	var args []string
	if ctx.IsDarwinPlatform() {
		stripBin = ctx.ShellCmd("xcodebuild -find strip")
		args = []string{"-x"}
	} else {
		stripBin = ctx.GetNDKToolchainBinPath("llvm-strip")
	}

	ctx.Tunnel.Spawn(&j9.SpawnOpt{
		Name: stripBin,
		Args: append(args, src, "-o", dst),
	})
}

func (ctx *BuildContext) getNDKClangPath(cpp bool) string {
	binName := GetOldArch(ctx.Arch) + "-linux-android" + MinAndroidAPI + "-clang"
	if cpp {
		binName += "++"
	}
	return ctx.GetNDKToolchainBinPath(binName)
}

type GetCompilerConfigureEnvOptions struct {
	// This might override other flags provided by source repo.
	// It's recommended to use `--extra-xxxflags` during `./configure`.
	OverrideCompilerFlags bool
}

func (ctx *BuildContext) GetCompilerConfigureEnv(opt *GetCompilerConfigureEnvOptions) []string {
	if opt == nil {
		opt = &GetCompilerConfigureEnvOptions{}
	}

	args := []string{
		"CC=" + ctx.GetCCPath(),
		"CXX=" + ctx.GetCXXPath(),
		"LD=" + ctx.GetLDPath(),
	}
	if ctx.IsAndroidPlatform() {
		args = append(args, "AR="+ctx.GetNDKToolchainBinPath("llvm-ar"))
		args = append(args, "AS="+ctx.GetNDKToolchainBinPath("llvm-as"))
		args = append(args, "RANLIB="+ctx.GetNDKToolchainBinPath("llvm-ranlib"))
		args = append(args, "STRIP="+ctx.GetNDKToolchainBinPath("llvm-strip"))
		args = append(args, "NM="+ctx.GetNDKToolchainBinPath("llvm-nm"))
	}

	if opt.OverrideCompilerFlags {
		cflags := ctx.GetCompilerFlags(nil)
		ldflags := ctx.GetCompilerFlags(&GetCompilerFlags{LD: true})

		args = append(args, "CFLAGS="+cflags)
		args = append(args, "CXXFLAGS="+cflags)
		args = append(args, "LDFLAGS="+ldflags)
	}

	return args
}

func (ctx *BuildContext) ShellCmd(cmd string) string {
	output := ctx.Tunnel.Shell(&j9.ShellOpt{
		Cmd: cmd})
	return strings.TrimSpace(string(output))
}

func (ctx *BuildContext) lipoStaticLibArch(file string) ArchEnum {
	output := ctx.ShellCmd(fmt.Sprintf("lipo -archs %s", file))
	switch output {
	case "arm64":
		return ArchArm64
	case "x86_64":
		return ArchX86_64
	default:
		panic(fmt.Sprintf("Unexpected arch: %s", output))
	}
}

func (ctx *BuildContext) androidReadStaticLibArch(file string) ArchEnum {
	ndkReadelf := ctx.GetNDKToolchainBinPath("llvm-readelf")
	output := ctx.ShellCmd(fmt.Sprintf("%s -h %s | grep -m1 Machine", ndkReadelf, file))
	// Sample output:
	//   Machine:                           AArch64
	colonIdx := strings.Index(output, ":")
	if colonIdx == -1 {
		panic(fmt.Sprintf("Cannot find Machine in %s", output))
	}
	archStr := strings.TrimSpace(output[colonIdx+1:])
	switch archStr {
	case "AArch64":
		return ArchArm64
	case "x86_64":
		fallthrough
	case "Advanced Micro Devices X86-64":
		return ArchX86_64
	default:
		panic(fmt.Sprintf("Unexpected arch: %s", archStr))
	}
}

func (ctx *BuildContext) CheckLocalStaticLibArch(fileName string) {
	var actualArch ArchEnum
	file := filepath.Join(ctx.LibsLibDir, fileName)
	if ctx.IsDarwinPlatform() {
		actualArch = ctx.lipoStaticLibArch(file)
	} else if ctx.IsAndroidPlatform() {
		actualArch = ctx.androidReadStaticLibArch(file)
	}
	if actualArch != ctx.Arch {
		panic(fmt.Sprintf("Unexpected arch: %s, expected: %s for file %s", actualArch, ctx.Arch, file))
	}
}

func (ctx *BuildContext) checkStaticLibMinSDKVer(file string, minSDKVer string) {
	output := ctx.ShellCmd(fmt.Sprintf("otool -l %s | grep -m 1 minos", file))
	// output looks like:
	// minos 18.2
	if strings.HasPrefix(output, "minos ") {
		verStr := strings.TrimPrefix(output, "minos ")
		if verStr != minSDKVer {
			panic(fmt.Sprintf("Unexpected min SDK version: %s, expected: %s for file %s", verStr, minSDKVer, file))
		}
		// Success,
		return
	}
	panic(fmt.Sprintf("Cannot find minos in %s", file))
}

func (ctx *BuildContext) MinDarwinSDKVer() string {
	switch ctx.SDK {
	case SDKMacos:
		return MinMacosVersion
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		return MinIosVersion
	}
	panic(ctx.UnsupportedError())
}

func (ctx *BuildContext) CheckLocalStaticLibMinSDKVer(fileName string) {
	file := filepath.Join(ctx.LibsLibDir, fileName)
	if ctx.IsDarwinPlatform() {
		ctx.checkStaticLibMinSDKVer(file, ctx.MinDarwinSDKVer())
	}
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

func (ctx *BuildContext) UnsupportedError() error {
	return fmt.Errorf("unsupported config. SDK: %s, Arch: %s", ctx.SDK, ctx.Arch)
}

func (ctx *BuildContext) CommonCmakeArgs(libType LibType) []string {
	var isDylib bool
	if SupportedLibTypes[libType] {
		isDylib = libType == LibTypeDynamic
	} else {
		panic(fmt.Sprintf("Invalid libType: %s, valid types: %v", libType, SupportedLibTypes))
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
	ctx.Tunnel.Logger().Log(j9.LogLevelVerbose, "[Cmake] Target OS: "+targetOS)

	args := []string{
		"-DCMAKE_SYSTEM_NAME=" + targetOS,
		"-DCMAKE_INSTALL_PREFIX=" + ctx.LibsDir,
		"-DCMAKE_PREFIX_PATH=" + ctx.LibsDir,
		"-DCMAKE_LIBRARY_PATH=" + ctx.LibsLibDir,
		"-DCMAKE_FIND_USE_CMAKE_SYSTEM_PATH=0",
		"-DCMAKE_FIND_USE_SYSTEM_ENVIRONMENT_PATH=0",
		// Always enable PIC regardless of lib type.
		"-DCMAKE_POSITION_INDEPENDENT_CODE=1",
	}

	isDylibStr := "0"
	if isDylib {
		isDylibStr = "1"
	}
	args = append(args, "-DBUILD_SHARED_LIBS="+isDylibStr)

	if ctx.IsDarwinPlatform() {
		args = append(args,
			// SDK
			"-DCMAKE_OSX_SYSROOT="+ctx.GetSDKPath(),
			// Min SDK
			"-DCMAKE_OSX_DEPLOYMENT_TARGET="+ctx.MinDarwinSDKVer(),
			// -arch
			"-DCMAKE_OSX_ARCHITECTURES="+string(ctx.Arch),
			"-DCMAKE_MACOSX_BUNDLE=0",
			"-DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_ALLOWED=0",
			// On Android, this should be set by `DCMAKE_TOOLCHAIN_FILE`.
			"-DCMAKE_SYSTEM_PROCESSOR="+string(ctx.Arch),
		)
	}

	if ctx.IsAndroidPlatform() {
		ndk := ctx.GetNDKPath()
		abi := GetABI(ctx.Arch)
		args = append(args,
			"-DANDROID_NDK="+ndk,
			"-DANDROID_ABI="+abi,
			"-DANDROID_PLATFORM=android-"+MinAndroidAPI,
			"-DCMAKE_ANDROID_NDK="+ndk,
			"-DCMAKE_TOOLCHAIN_FILE="+ctx.GetNDKCmakeToolchainFile(),
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

func (ctx *BuildContext) GetAutoconfHost() string {
	switch ctx.SDK {
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		switch ctx.Arch {
		case ArchArm64:
			return "arm64-apple-darwin"
		case ArchX86_64:
			return "x86_64-apple-darwin"
		default:
			return ""
		}
	case SDKMacos:
		switch ctx.Arch {
		case ArchArm64:
			return "arm64-apple-darwin"
		case ArchX86_64:
			return "x86_64-apple-darwin"
		}
	case SDKAndroid:
		return GetOldArch(ctx.Arch) + "-linux-android" + MinAndroidAPI
	}
	return ""
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
		"KU_LIBS_DIR=" + ctx.LibsDir,
		"KU_LIBS_INCLUDE_DIR=" + ctx.LibsIncludeDir,
		"KU_LIBS_LIB_DIR=" + ctx.LibsLibDir,
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

func (ctx *BuildContext) MustGetAutoconfHost() string {
	host := ctx.GetAutoconfHost()
	if host == "" {
		panic(ctx.UnsupportedError())
	}
	return host
}

type cacheStringFunc func() string

func (ctx *BuildContext) cacheSDKArchString(key string, fn cacheStringFunc) string {
	if ctx.stringCache == nil {
		ctx.stringCache = map[string]string{}
	}
	key = string(ctx.SDK) + "_" + string(ctx.Arch) + "_" + key
	if val, ok := ctx.stringCache[key]; ok {
		return val
	}
	val := fn()
	ctx.stringCache[key] = val
	return val
}
