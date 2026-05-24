package ku

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// When set, prevents automatic cleaning of the output directory before each loop iteration.
	DisableAutoClean bool
}

func StartEnvLoopWithOptions(cliOpt *CLIOptions, opt *StartEnvLoopOptions) {
	if opt == nil || opt.LoopFn == nil {
		panic("StartEnvLoopWithOptions: LoopFn is required")
	}
	cliArgs := ParseCLIArgs(cliOpt)
	tunnel := CreateDefaultTunnel()
	shell := NewShell(tunnel, cliArgs)

	if opt.BeforeAllFn != nil {
		shell.Log(j9.LogLevelInfo, "🚕 Running BeforeAllFn")
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
			shell.Log(j9.LogLevelInfo, fmt.Sprintf("🚕 Running loop for SDK=%s, Arch=%s", sdk, arch))
			osEnv := NewOSEnv(shell, sdk, arch)
			env := NewBuildEnv(shell, osEnv)

			if !opt.DisableAutoClean {
				shell.Log(j9.LogLevelInfo, fmt.Sprintf("🚕 Cleaning output directory: %s", env.OutDir))
				io2.CleanDir(env.OutDir)
			}
			opt.LoopFn(env)
		}
	}

	if opt.AfterAllFn != nil {
		shell.Log(j9.LogLevelInfo, "🚕 Running AfterAllFn")
		opt.AfterAllFn(shell)
	}

	shell.Log(j9.LogLevelInfo, "🚕 Build loop completed")
}

func StartEnvLoop(cliOpt *CLIOptions, fn func(*BuildEnv)) {
	StartEnvLoopWithOptions(cliOpt, &StartEnvLoopOptions{
		LoopFn: fn,
	})
}

func GetTargetDistDir(targetDir string) string {
	distDir := filepath.Join(targetDir, DistDirName)
	return distDir
}

func CopyJNILibs(shell *Shell, libFileNames []string, headerFileNames []string) {
	cliArgs := shell.Args
	target := cliArgs.Target
	debug := cliArgs.DebugBuild
	buildTypeDir := GetBuildTypeDir(debug)
	sdkDir := GetSDKDir(buildTypeDir, SDKAndroid)

	jniBuildDir := filepath.Join(sdkDir, "jni", target)
	dstLibsDir := filepath.Join(jniBuildDir, "jniLibs")
	dstIncludeDir := filepath.Join(jniBuildDir, "include")

	CopyJNILibsCore(&CopyJNILibsOptions{
		Shell:           shell,
		DstLibsDir:      dstLibsDir,
		DstIncludeDir:   dstIncludeDir,
		LibFileNames:    libFileNames,
		HeaderFileNames: headerFileNames,
		Target:          target,
		Debug:           debug,
	})
}

type CopyJNILibsOptions struct {
	Shell           *Shell
	DstLibsDir      string
	DstIncludeDir   string
	LibFileNames    []string
	HeaderFileNames []string
	Target          string
	Debug           bool
	KuDeploy        bool
}

func CopyJNILibsCore(opt *CopyJNILibsOptions) {
	if opt == nil {
		panic("CopyJNILibsCore: options cannot be nil")
	}
	shell := opt.Shell
	dstLibsDir := opt.DstLibsDir
	dstIncludeDir := opt.DstIncludeDir
	libFileNames := opt.LibFileNames
	headerFileNames := opt.HeaderFileNames
	target := opt.Target
	debug := opt.Debug

	buildTypeDir := GetBuildTypeDir(debug)
	sdkDir := GetSDKDir(buildTypeDir, SDKAndroid)

	if opt.KuDeploy {
		shell.Log(j9.LogLevelInfo, fmt.Sprintf("Copying JNI libs for target=%s, debug=%v, dstLibsDir=%s, dstIncludeDir=%s, libFileNames=%v, headerFileNames=%v", target, debug, dstLibsDir, dstIncludeDir, libFileNames, headerFileNames))
	}

	io2.Mkdirp(dstLibsDir)
	if dstIncludeDir != "" {
		io2.Mkdirp(dstIncludeDir)
	}

	// Copy the JNI libs to the jniLibs directory.
	jniArchList := []ArchEnum{
		ArchArm64,
		ArchX86_64,
	}

	for _, arch := range jniArchList {
		for _, libFileName := range libFileNames {
			archDir := GetSDKArchDir(sdkDir, arch)
			targetDir := filepath.Join(archDir, target)
			targetDistDir := GetTargetDistDir(targetDir)
			srcLibFile := filepath.Join(targetDistDir, "lib", libFileName)
			if !strings.HasSuffix(srcLibFile, ".so") {
				srcLibFile += ".so"
			}

			var jniArch string
			if arch == ArchArm64 {
				jniArch = "arm64-v8a"
			} else {
				jniArch = "x86_64"
			}
			jniArchDir := filepath.Join(dstLibsDir, jniArch)
			io2.Mkdirp(jniArchDir)

			// Copy the lib file to the jniLibs directory.
			CPToDirByForce(shell, srcLibFile, false, jniArchDir)

			if opt.KuDeploy {
				shell.Log(j9.LogLevelInfo, fmt.Sprintf("✅ Deployed %s to %s", libFileName, jniArchDir))
			}
		}
	}

	arm64ArchDir := GetSDKArchDir(sdkDir, ArchArm64)
	arm64TargetDir := filepath.Join(arm64ArchDir, target)
	arm64TargetDistDir := GetTargetDistDir(arm64TargetDir)
	headerSrcDir := filepath.Join(arm64TargetDistDir, "include")
	for _, headerFileName := range headerFileNames {
		srcHeaderFile := filepath.Join(headerSrcDir, headerFileName)

		// Copy the header file to the include directory.
		isSrcDir := io2.DirectoryExists(srcHeaderFile)
		CPToDirByForce(shell, srcHeaderFile, isSrcDir, dstIncludeDir)
	}
}

func CPToDirByForce(shell *Shell, src string, isSrcDir bool, dstDir string) {
	if !strings.HasSuffix(dstDir, "/") {
		dstDir += "/"
	}
	if isSrcDir {
		dstPath := filepath.Join(dstDir, filepath.Base(src))
		if io2.DirectoryExists(dstPath) {
			err := os.RemoveAll(dstPath)
			if err != nil {
				shell.Quit(fmt.Sprintf("Error removing existing directory: %v\n", err))
			}
		}

		shell.Spawn(&j9.SpawnOpt{
			Name: "cp",
			Args: []string{"-R", src, dstDir}},
		)
		return
	}

	// When cp a file into a dir, cp will overwrite the file if it already exists, so we don't need to delete it first.
	shell.Spawn(&j9.SpawnOpt{
		Name: "cp",
		Args: []string{src, dstDir}},
	)
}
