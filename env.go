package ku

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
	"github.com/mgenware/ku-builder/util"
)

// Env = SDK + Arch.
type Env struct {
	CLIArgs *CLIArgs
	SDK     SDKEnum
	Arch    ArchEnum

	shell       *Shell
	stringCache *util.StringCache
}

func NewEnv(shell *Shell, cliArgs *CLIArgs, sdk SDKEnum, arch ArchEnum) *Env {
	return &Env{
		CLIArgs:     cliArgs,
		shell:       shell,
		SDK:         sdk,
		Arch:        arch,
		stringCache: util.NewStringCache(),
	}
}

func (e *Env) IsIosPlatform() bool {
	return e.SDK == SDKIos || e.SDK == SDKIosSimulator
}

func (e *Env) IsDarwinPlatform() bool {
	return e.SDK == SDKMacos || e.IsIosPlatform()
}

func (e *Env) IsAndroidPlatform() bool {
	return e.SDK == SDKAndroid
}

func (e *Env) GetSDKRootPath() string {
	return e.cachedString("sdk-root", func() string {
		return io2.DirectoryMustExist(e.fetchSDKRootPath())
	})
}

func (e *Env) fetchSDKRootPath() string {
	switch e.SDK {
	case SDKMacos:
		return e.shell.Shell("xcrun --sdk macosx --show-sdk-path")
	case SDKIos:
		return e.shell.Shell("xcrun --sdk iphoneos --show-sdk-path")
	case SDKIosSimulator:
		return e.shell.Shell("xcrun --sdk iphonesimulator --show-sdk-path")
	case SDKAndroid:
		return filepath.Join(e.getNDKToolchainRootPath(), "sysroot")
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetCCPath() string {
	return e.cachedString("cc", e.fetchCCPath)
}

func (e *Env) fetchCCPath() string {
	if e.IsDarwinPlatform() {
		return e.shell.Shell("xcodebuild -find clang")
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(false)
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetCXXPath() string {
	return e.cachedString("cxx", e.fetchCXXPath)
}

func (e *Env) fetchCXXPath() string {
	if e.IsDarwinPlatform() {
		return e.shell.Shell("xcodebuild -find clang++")
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(true)
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetLDPath() string {
	return e.cachedString("ld", e.fetchLDPath)
}

func (e *Env) fetchLDPath() string {
	if e.IsDarwinPlatform() {
		return e.GetCCPath()
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(false)
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetAndroidSDKPath() string {
	if e.IsAndroidPlatform() {
		return e.cachedString("android_sdk", func() string {
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
	panic(e.UnsupportedError())
}

func (e *Env) GetNDKPath() string {
	if e.IsAndroidPlatform() {
		return e.cachedString("ndk", func() string {
			path := os.Getenv("ANDROID_NDK_PATH")
			if path != "" {
				return path
			}
			path = e.CLIArgs.NDK
			// If `NDKInput` is not an absolute path, it's considered an NDK version.
			if !strings.HasPrefix(path, "/") {
				path = filepath.Join(e.GetAndroidSDKPath(), "ndk", path)
			}
			io2.DirectoryMustExist(path)
			return path
		})
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetNDKCmakeToolchainFile() string {
	if e.IsAndroidPlatform() {
		return e.cachedString("ndk_cmake_toolchain", func() string {
			path := filepath.Join(e.GetNDKPath(), "build/cmake/android.toolchain.cmake")
			io2.FileMustExist(path)
			return path
		})
	}
	panic(e.UnsupportedError())
}

func (e *Env) GetDylibExt() string {
	if e.IsDarwinPlatform() {
		return ".dylib"
	}
	return ".so"
}

func (e *Env) getNDKToolchainRootPath() string {
	ndkPath := e.GetNDKPath()
	return e.cachedString("ndk-toolchain-root", func() string {
		path := filepath.Join(ndkPath, "toolchains", "llvm", "prebuilt", "darwin-x86_64")
		return io2.DirectoryMustExist(path)
	})
}

func (e *Env) GetNDKToolchainBinPath(name string) string {
	return e.cachedString("ndk-toolchain-bin-"+name, func() string {
		path := filepath.Join(e.getNDKToolchainRootPath(), "bin", name)
		return io2.FileMustExist(path)
	})
}

func (e *Env) StripFile(src, dst string) {
	var stripBin string
	var args []string
	if e.IsDarwinPlatform() {
		stripBin = e.shell.Shell("xcodebuild -find strip")
		args = []string{"-x"}
	} else {
		stripBin = e.GetNDKToolchainBinPath("llvm-strip")
	}

	e.shell.Spawn(&j9.SpawnOpt{
		Name: stripBin,
		Args: append(args, src, "-o", dst),
	})
}

func (e *Env) getNDKClangPath(cpp bool) string {
	binName := GetOldArch(e.Arch) + "-linux-android" + MinAndroidAPI + "-clang"
	if cpp {
		binName += "++"
	}
	return e.GetNDKToolchainBinPath(binName)
}

func (e *Env) UnsupportedError() error {
	return fmt.Errorf("unsupported environment. SDK: %s, Arch: %s", e.SDK, e.Arch)
}

func (e *Env) lipoStaticLibArch(file string) ArchEnum {
	output := e.shell.Shell(fmt.Sprintf("lipo -archs %s", file))
	switch output {
	case "arm64":
		return ArchArm64
	case "x86_64":
		return ArchX86_64
	default:
		panic(fmt.Sprintf("Unexpected arch: %s", output))
	}
}

func (e *Env) androidReadStaticLibArch(file string) ArchEnum {
	ndkReadelf := e.GetNDKToolchainBinPath("llvm-readelf")
	output := e.shell.Shell(fmt.Sprintf("%s -h %s | grep -m1 Machine", ndkReadelf, file))
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

func (e *Env) CheckLocalStaticLibArch(file string) {
	var actualArch ArchEnum
	if e.IsDarwinPlatform() {
		actualArch = e.lipoStaticLibArch(file)
	} else if e.IsAndroidPlatform() {
		actualArch = e.androidReadStaticLibArch(file)
	}
	if actualArch != e.Arch {
		panic(fmt.Sprintf("Unexpected arch: %s, expected: %s for file %s", actualArch, e.Arch, file))
	}
}

func (e *Env) checkStaticLibMinSDKVer(file string, minSDKVer string) {
	output := e.shell.Shell(fmt.Sprintf("otool -l %s | grep -m 1 minos", file))
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

func (e *Env) MinDarwinSDKVer() string {
	switch e.SDK {
	case SDKMacos:
		return MinMacosVersion
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		return MinIosVersion
	}
	panic(e.UnsupportedError())
}

func (e *Env) CheckLocalStaticLibMinSDKVer(file string) {
	if e.IsDarwinPlatform() {
		e.checkStaticLibMinSDKVer(file, e.MinDarwinSDKVer())
	}
}

func (e *Env) GetAutoconfHost() string {
	switch e.SDK {
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		switch e.Arch {
		case ArchArm64:
			return "arm64-apple-darwin"
		case ArchX86_64:
			return "x86_64-apple-darwin"
		default:
			return ""
		}
	case SDKMacos:
		switch e.Arch {
		case ArchArm64:
			return "arm64-apple-darwin"
		case ArchX86_64:
			return "x86_64-apple-darwin"
		}
	case SDKAndroid:
		return GetOldArch(e.Arch) + "-linux-android" + MinAndroidAPI
	}
	return ""
}

func (e *Env) MustGetAutoconfHost() string {
	host := e.GetAutoconfHost()
	if host == "" {
		panic(e.UnsupportedError())
	}
	return host
}

func (e *Env) cachedString(key string, fn util.StringCacheGetFn) string {
	key = string(e.SDK) + "_" + string(e.Arch) + "_" + key
	return e.stringCache.Get(key, fn)
}
