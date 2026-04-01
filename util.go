package ku

import (
	"path/filepath"
	"slices"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type StartEnvLoopOptions struct {
	// The main function to execute for each SDK/arch combination.
	LoopFn func(*BuildEnv)
	// Called before the loop starts, can be used for setup.
	BeforeAllFn func(*Shell)
	// Called after the loop ends, can be used for teardown.
	AfterAllFn func(*Shell)
	// When set, verifies file archs in the dist/lib directory after the loop.
	VerifyDistLibFileArch []string
	// When set, prevents automatic cleaning of the output directory before each loop iteration.
	DisableAutoClean bool
}

func StartEnvLoopWithOptions(cliOpt *CLIOptions, opt *StartEnvLoopOptions) {
	if opt == nil || opt.LoopFn == nil {
		panic("StartLoopWithOptions: LoopFn is required")
	}
	cliArgs := ParseCLIArgs(cliOpt)
	tunnel := CreateDefaultTunnel()
	shell := NewShell(tunnel, cliArgs)

	if opt.BeforeAllFn != nil {
		opt.BeforeAllFn(shell)
	}

	for _, sdk := range cliArgs.SDKs {
		var archs []ArchEnum
		if cliArgs.Arch != "" {
			archs = append(archs, cliArgs.Arch)
		} else {
			archs = SDKArchs[sdk]
		}

		for _, arch := range archs {
			osEnv := NewOSEnv(shell, sdk, arch)
			env := NewBuildEnv(shell, osEnv)

			if !opt.DisableAutoClean {
				io2.CleanDir(env.OutDir)
			}
			opt.LoopFn(env)

			if len(opt.VerifyDistLibFileArch) > 0 {
				env.VerifyDistLibFileArch(opt.VerifyDistLibFileArch)
			}
		}
	}

	if opt.AfterAllFn != nil {
		opt.AfterAllFn(shell)
	}
}

func StartEnvLoop(cliOpt *CLIOptions, fn func(*BuildEnv)) {
	StartEnvLoopWithOptions(cliOpt, &StartEnvLoopOptions{
		LoopFn: fn,
	})
}

func GetTargetDistDir(targetDir string) string {
	// Dist dir could be ${TargetDir}/dist or ${TargetDir}/libs.
	distDir := filepath.Join(targetDir, DistDirName)
	if io2.DirectoryExists(distDir) {
		return distDir
	}
	distDir = filepath.Join(targetDir, OutDirName)
	return distDir
}

func CopyJNILibs(shell *Shell, libFileNames []string, headerFileNames []string) {
	cliArgs := shell.Args
	if !slices.Contains(cliArgs.SDKs, SDKAndroid) {
		return
	}
	buildTypeDir := GetBuildTypeDir(cliArgs.DebugBuild)
	sdkDir := GetSDKDir(buildTypeDir, SDKAndroid)

	jniBuildDir := filepath.Join(sdkDir, "jni", cliArgs.Target)
	libsDir := filepath.Join(jniBuildDir, "jniLibs")
	includeDir := filepath.Join(jniBuildDir, "include")

	io2.Mkdirp(libsDir)
	io2.Mkdirp(includeDir)

	// Copy the JNI libs to the jniLibs directory.
	jniArchList := []ArchEnum{
		ArchArm64,
		ArchX86_64,
	}

	for _, arch := range jniArchList {
		for _, libFileName := range libFileNames {
			archDir := GetSDKArchDir(sdkDir, arch)
			targetDir := filepath.Join(archDir, cliArgs.Target)
			targetDistDir := GetTargetDistDir(targetDir)
			srcLibFile := filepath.Join(targetDistDir, "lib", libFileName)

			var jniArch string
			if arch == ArchArm64 {
				jniArch = "arm64-v8a"
			} else {
				jniArch = "x86_64"
			}
			jniArchDir := filepath.Join(libsDir, jniArch)
			io2.Mkdirp(jniArchDir)

			// Copy the lib file to the jniLibs directory.
			shell.Spawn(&j9.SpawnOpt{
				Name: "cp",
				Args: []string{srcLibFile, jniArchDir + "/"}},
			)
		}
	}

	arm64ArchDir := GetSDKArchDir(sdkDir, ArchArm64)
	arm64TargetDir := filepath.Join(arm64ArchDir, cliArgs.Target)
	arm64TargetDistDir := GetTargetDistDir(arm64TargetDir)
	headerSrcDir := filepath.Join(arm64TargetDistDir, "include")
	for _, headerFileName := range headerFileNames {
		srcHeaderFile := filepath.Join(headerSrcDir, headerFileName)

		// Copy the header file to the include directory.
		shell.Spawn(&j9.SpawnOpt{
			Name: "cp",
			Args: []string{"-R", srcHeaderFile, includeDir + "/"}},
		)
	}
}
