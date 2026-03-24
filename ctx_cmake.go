package ku

import (
	"fmt"
	"runtime"

	"github.com/mgenware/j9/v3"
)

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

	ctx.VerifyOutLibFileArch(outFile)
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

type CommonCmakeArgsOptions struct {
	EnableSystemPath bool
	DisablePIC       bool
}

func (ctx *BuildContext) CommonCmakeGenArgs(libType LibType) []string {
	return ctx.CommonCmakeGenArgsWithOptions(libType, nil)
}

func (ctx *BuildContext) CommonCmakeGenArgsWithOptions(libType LibType, opt *CommonCmakeArgsOptions) []string {
	if opt == nil {
		opt = &CommonCmakeArgsOptions{}
	}

	var isDylib bool
	if SupportedLibTypes[libType] {
		isDylib = libType == LibTypeDynamic
	} else {
		ctx.Shell.Quit(fmt.Sprintf("Invalid libType: %v, valid types: %v", libType, SupportedLibTypes))
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
