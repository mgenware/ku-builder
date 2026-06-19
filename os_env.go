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

var gStringCache = util.NewStringCache()

// OSEnv = SDK + Arch.
type OSEnv struct {
	SDK  SDKEnum
	Arch ArchEnum

	shell       *Shell
	stringCache *util.StringCache
}

func NewOSEnv(shell *Shell, sdk SDKEnum, arch ArchEnum) *OSEnv {
	return &OSEnv{
		shell:       shell,
		SDK:         sdk,
		Arch:        arch,
		stringCache: util.NewStringCache(),
	}
}

func (e *OSEnv) IsIosPlatform() bool {
	return e.SDK == SDKIos || e.SDK == SDKIosSimulator
}

func (e *OSEnv) IsDarwinPlatform() bool {
	return e.SDK == SDKMacos || e.IsIosPlatform()
}

func (e *OSEnv) IsAndroidPlatform() bool {
	return e.SDK == SDKAndroid
}

func (e *OSEnv) GetSDKRootPath() string {
	return e.cachedString("sdk-root", func() string {
		return io2.DirectoryMustExist(e.fetchSDKRootPath())
	})
}

func (e *OSEnv) fetchSDKRootPath() string {
	switch e.SDK {
	case SDKMacos:
		return e.shell.ShellCached("xcrun --sdk macosx --show-sdk-path")
	case SDKIos:
		return e.shell.ShellCached("xcrun --sdk iphoneos --show-sdk-path")
	case SDKIosSimulator:
		return e.shell.ShellCached("xcrun --sdk iphonesimulator --show-sdk-path")
	case SDKAndroid:
		return filepath.Join(e.getNDKToolchainRootPath(), "sysroot")
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetCCPath() string {
	return e.cachedString("cc", e.fetchCCPath)
}

func (e *OSEnv) fetchCCPath() string {
	if e.IsDarwinPlatform() {
		return e.RunXcodeFindCached("clang")
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(false)
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) RunXcodeFindCached(name string) string {
	return e.shell.ShellCached("xcodebuild -find " + name)
}

func (e *OSEnv) GetCXXPath() string {
	return e.cachedString("cxx", e.fetchCXXPath)
}

func (e *OSEnv) fetchCXXPath() string {
	if e.IsDarwinPlatform() {
		return e.RunXcodeFindCached("clang++")
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(true)
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetLDPath() string {
	return e.cachedString("ld", e.fetchLDPath)
}

func (e *OSEnv) GetSDKArchString() string {
	return string(e.SDK) + "-" + string(e.Arch)
}

func (e *OSEnv) GetPkgConfigPath() string {
	return e.GetWhichExe("pkg-config")
}

func (e *OSEnv) GetMakePath() string {
	return e.GetWhichExe("make")
}

func (e *OSEnv) GetWhichExe(name string) string {
	return e.shell.ShellCached("which " + name)
}

func (e *OSEnv) fetchLDPath() string {
	if e.IsDarwinPlatform() {
		return e.GetCCPath()
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(false)
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetAndroidSDKPath() string {
	if e.IsAndroidPlatform() {
		return globalCachedString("android_sdk", func() string {
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
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetNDKPath() string {
	if e.IsAndroidPlatform() {
		return globalCachedString("ndk-path", func() string {
			path := os.Getenv("ANDROID_NDK_PATH")
			if path != "" {
				return path
			}
			path = e.shell.Args.NDK
			// If `NDKInput` is not an absolute path, it's considered an NDK version.
			if !strings.HasPrefix(path, "/") {
				path = filepath.Join(e.GetAndroidSDKPath(), "ndk", path)
			}
			io2.DirectoryMustExist(path)
			return path
		})
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetNDKCmakeToolchainFile() string {
	if e.IsAndroidPlatform() {
		return globalCachedString("ndk_cmake_toolchain", func() string {
			path := filepath.Join(e.GetNDKPath(), "build/cmake/android.toolchain.cmake")
			io2.FileMustExist(path)
			return path
		})
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) LibTypeExt(libType LibType) string {
	if libType == LibTypeStatic {
		return ".a"
	}
	if e.IsDarwinPlatform() {
		return ".dylib"
	}
	if e.IsAndroidPlatform() {
		return ".so"
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) getNDKToolchainRootPath() string {
	ndkPath := e.GetNDKPath()
	return globalCachedString("ndk-toolchain-root", func() string {
		path := filepath.Join(ndkPath, "toolchains", "llvm", "prebuilt", "darwin-x86_64")
		return io2.DirectoryMustExist(path)
	})
}

func (e *OSEnv) GetNDKToolchainBinPath(name string) string {
	return globalCachedString("ndk-toolchain-bin-"+name, func() string {
		path := filepath.Join(e.getNDKToolchainRootPath(), "bin", name)
		return io2.FileMustExist(path)
	})
}

func (e *OSEnv) StripFile(src, dst string) {
	var stripBin string
	var args []string
	if e.IsDarwinPlatform() {
		stripBin = e.RunXcodeFindCached("strip")
		args = []string{"-x"}
	} else {
		stripBin = e.GetNDKToolchainBinPath("llvm-strip")
	}

	e.shell.Spawn(&j9.SpawnOpt{
		Name: stripBin,
		Args: append(args, src, "-o", dst),
	})
}

func (e *OSEnv) getNDKClangPath(cpp bool) string {
	binName := GetOldArch(e.Arch) + "-linux-android" + MinAndroidAPI + "-clang"
	if cpp {
		binName += "++"
	}
	return e.GetNDKToolchainBinPath(binName)
}

//go:noreturn
func (e *OSEnv) ThrowUnsupportedError() error {
	e.shell.Quit(fmt.Sprintf("unsupported environment. SDK: %s, Arch: %s", e.SDK, e.Arch))
	panic("unreachable")
}

func (e *OSEnv) readDarwinLibArch(file string) ArchEnum {
	output := e.shell.Shell(fmt.Sprintf("lipo -archs %s", file))
	switch output {
	case "arm64":
		return ArchArm64
	case "x86_64":
		return ArchX86_64
	default:
		e.shell.Quit(fmt.Sprintf("Unexpected arch: %s", output))
		panic("unreachable")
	}
}

func (e *OSEnv) readAndroidLibArch(file string) ArchEnum {
	ndkReadelf := e.GetNDKToolchainBinPath("llvm-readelf")
	output := e.shell.Shell(fmt.Sprintf("%s -h %s | grep -m1 Machine", ndkReadelf, file))
	// Sample output:
	//   Machine:                           AArch64
	colonIdx := strings.Index(output, ":")
	if colonIdx == -1 {
		e.shell.Quit(fmt.Sprintf("Cannot find Machine in %s", output))
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
		e.shell.Quit(fmt.Sprintf("Unexpected arch: %s", archStr))
		panic("unreachable")
	}
}

func (e *OSEnv) VerifyFileArch(libType LibType, file string) {
	logger := e.shell.Logger()

	logger.Log(j9.LogLevelVerbose, "🔍 Verifying arch for file "+file)
	var actualArch ArchEnum
	if e.IsDarwinPlatform() {
		actualArch = e.readDarwinLibArch(file)
	} else if e.IsAndroidPlatform() {
		actualArch = e.readAndroidLibArch(file)
	}

	if actualArch != e.Arch {
		logger.Log(j9.LogLevelError, fmt.Sprintf("Arch mismatch for file %s, expected: %s, actual: %s", file, e.Arch, actualArch))
		os.Exit(1)
	} else {
		logger.Log(j9.LogLevelSuccess, fmt.Sprintf("✅ Arch verified for file %s, expected: %s", file, e.Arch))
	}
}

func (e *OSEnv) GetDarwinClangTargetTriple() string {
	if !e.IsDarwinPlatform() {
		e.ThrowUnsupportedError()
	}
	archStr := string(e.Arch)
	switch e.SDK {
	case SDKMacos:
		return archStr + "-apple-macosx" + MinMacosVersion
	case SDKIosSimulator:
		return archStr + "-apple-ios" + MinIosVersion + "-simulator"
	case SDKIos:
		return archStr + "-apple-ios" + MinIosVersion
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) VerifyDarwinStaticLibSDK(file string, minSDKVer string, sdk SDKEnum) {
	if !e.IsDarwinPlatform() {
		e.ThrowUnsupportedError()
	}

	output := e.shell.Shell(fmt.Sprintf("otool -l %s | grep -E 'platform|minos'", file))
	// Sample output:
	//  platform 2
	//  minos 14.0

	lines := strings.Split(output, "\n")
	platformVerified := false
	versionVerified := false

	otoolPlatform := ""
	switch sdk {
	case SDKMacos:
		otoolPlatform = "1"
	case SDKIos:
		otoolPlatform = "2"
	case SDKIosSimulator:
		otoolPlatform = "7"
	default:
		e.shell.Quit(fmt.Sprintf("Unsupported SDK for otool verification: %s", sdk))
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) != 2 {
			e.shell.Quit(fmt.Sprintf("Unexpected line in otool output: %s", line))
		}
		key := parts[0]
		value := parts[1]

		if key == "platform" {
			if value != otoolPlatform {
				e.shell.Quit(fmt.Sprintf("Unexpected platform in otool output: %s, expected: %s for file %s", value, otoolPlatform, file))
			}
			platformVerified = true
		} else if key == "minos" {
			if value != minSDKVer {
				e.shell.Quit(fmt.Sprintf("Unexpected min SDK version in otool output: %s, expected: %s for file %s", value, minSDKVer, file))
			}
			versionVerified = true
		}
	}

	if !platformVerified {
		e.shell.Quit(fmt.Sprintf("platform not found in otool output for file %s, output: %s", file, output))
	}
	if !versionVerified {
		e.shell.Quit(fmt.Sprintf("minos not found in otool output for file %s, output: %s", file, output))
	}
}

func (e *OSEnv) MinDarwinSDKVer() string {
	switch e.SDK {
	case SDKMacos:
		return MinMacosVersion
	case SDKIos:
		fallthrough
	case SDKIosSimulator:
		return MinIosVersion
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetAutoconfHost() string {
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

func (e *OSEnv) MustGetAutoconfHost() string {
	host := e.GetAutoconfHost()
	if host == "" {
		e.ThrowUnsupportedError()
		panic("unreachable")
	}
	return host
}

func (e *OSEnv) cachedString(key string, fn util.StringCacheGetFn) string {
	return e.stringCache.Get(key, fn)
}

func globalCachedString(key string, fn util.StringCacheGetFn) string {
	return gStringCache.Get(key, fn)
}
