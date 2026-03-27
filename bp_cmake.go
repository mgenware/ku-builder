package ku

import (
	"fmt"
	"runtime"

	"github.com/mgenware/j9/v3"
)

type RunCmakeGenOptions struct {
	Args []string
	Env  []string
}

func (bp *BuildProject) RunCmakeGen(opt *RunCmakeGenOptions) {
	bp.NotNullOrQuit(opt, "opt")
	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(), opt.Env...)
	env = append(env,
		"KU_CMAKE_ACTION=gen",
	)

	bp.Shell.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: opt.Args,
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

func (bp *BuildProject) RunCmakeBuildOrInstall(opt *RunCmakeBuildOrInstallOptions, outFile []string) {
	bp.NotNullOrQuit(opt, "opt")
	bp.NotNullOrQuit(opt.Action, "opt.Action")

	args := []string{
		"--" + string(opt.Action), ".",
	}

	if opt.Target != "" {
		if opt.Action == CmakeActionInstall {
			bp.Shell.Quit("opt.Target is not supported for install")
		}
		args = append(args, "--target", opt.Target)
	}

	var config string
	cliArgs := bp.Shell.Args
	if cliArgs.DebugBuild {
		config = "Debug"
	} else {
		config = "Release"
	}
	args = append(args, "--config", config)

	// Strip during production install.
	if opt.Action == CmakeActionInstall && !cliArgs.DebugBuild {
		// This uses `CMAKE_STRIP`, which is set by Android toolchain.
		args = append(args, "--strip")
	}

	if opt.Action == CmakeActionBuild {
		numCores := runtime.NumCPU()
		args = append(args, "-j", fmt.Sprintf("%v", numCores))
	}

	// Extra args.
	if len(opt.ExtraArgs) > 0 {
		args = append(args, opt.ExtraArgs...)
	}

	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(), opt.Env...)
	env = append(env,
		"KU_CMAKE_ACTION="+string(opt.Action),
	)
	bp.Shell.Spawn(&j9.SpawnOpt{
		Name: "cmake",
		Args: args,
		Env:  env,
	})

	bp.BuildEnv.VerifyOutLibFileArch(outFile)
}

func (bp *BuildProject) RunCmakeBuild() {
	bp.RunCmakeBuildTarget("")
}

func (bp *BuildProject) RunCmakeBuildTarget(target string) {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: CmakeActionBuild,
		Target: target,
	}
	bp.RunCmakeBuildOrInstall(opt, nil)
}

func (bp *BuildProject) RunCmakeInstall(outFile []string) {
	opt := &RunCmakeBuildOrInstallOptions{
		Action: CmakeActionInstall,
	}
	bp.RunCmakeBuildOrInstall(opt, outFile)
}

type GetCmakeGenArgsOptions struct {
	EnableSystemPath bool
	DisablePIC       bool
	Preset           string
}

func (bp *BuildProject) GetCmakeGenArgs() []string {
	return bp.GetCmakeGenArgsWithOptions(nil)
}

func (bp *BuildProject) GetCmakeGenArgsWithOptions(opt *GetCmakeGenArgsOptions) []string {
	if opt == nil {
		opt = &GetCmakeGenArgsOptions{}
	}

	libType := bp.LibType
	var isDylib bool
	if SupportedLibTypes[libType] {
		isDylib = libType == LibTypeDynamic
	} else {
		bp.Shell.Quit(fmt.Sprintf("Invalid libType: %v, valid types: %v", libType, SupportedLibTypes))
	}

	var targetOS string
	osEnv := bp.OS
	buildEnv := bp.BuildEnv
	cliArgs := bp.Shell.Args

	switch osEnv.SDK {
	case SDKMacos:
		targetOS = "Darwin"
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		targetOS = "iOS"
	case SDKAndroid:
		targetOS = "Android"
	}

	args := []string{
		"-DCMAKE_SYSTEM_NAME=" + targetOS,
		"-DCMAKE_INSTALL_PREFIX=" + buildEnv.OutDir,
		"-DCMAKE_PREFIX_PATH=" + buildEnv.OutDir,
		"-DCMAKE_LIBRARY_PATH=" + buildEnv.OutLibDir,
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

	if osEnv.IsDarwinPlatform() {
		args = append(args,
			// SDK
			"-DCMAKE_OSX_SYSROOT="+osEnv.GetSDKRootPath(),
			// Min SDK
			"-DCMAKE_OSX_DEPLOYMENT_TARGET="+osEnv.MinDarwinSDKVer(),
			// -arch
			"-DCMAKE_OSX_ARCHITECTURES="+string(osEnv.Arch),
			"-DCMAKE_MACOSX_BUNDLE=0",
			"-DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_ALLOWED=0",
			// On Android, this should be set by `DCMAKE_TOOLCHAIN_FILE`.
			"-DCMAKE_SYSTEM_PROCESSOR="+string(osEnv.Arch),
		)
	}

	if osEnv.IsAndroidPlatform() {
		ndk := osEnv.GetNDKPath()
		abi := GetABI(osEnv.Arch)
		args = append(args,
			"-DANDROID_NDK="+ndk,
			"-DANDROID_ABI="+abi,
			"-DANDROID_PLATFORM=android-"+MinAndroidAPI,
			"-DCMAKE_ANDROID_NDK="+ndk,
			"-DCMAKE_TOOLCHAIN_FILE="+osEnv.GetNDKCmakeToolchainFile(),
			"-DCMAKE_ANDROID_ARCH_ABI="+abi,
			"-DCMAKE_SYSTEM_VERSION="+MinAndroidAPI,
		)
	}

	var buildType string
	if cliArgs.DebugBuild {
		buildType = "Debug"
	} else {
		buildType = "Release"
	}
	args = append(args, "-DCMAKE_BUILD_TYPE="+buildType)

	if cliArgs.CleanBuild {
		args = append(args, "--fresh")
	}
	if opt.Preset != "" {
		args = append(args, "--preset", opt.Preset)
	}

	// Put source and build dir arguments at the end.
	args = append(args, "-S", ".")
	args = append(args, "-B", bp.BuildDir)

	return args
}

func (bp *BuildProject) GoToBuildDir() {
	bp.Shell.CD(bp.BuildDir)
}
