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
		return e.shell.Shell("xcrun --sdk macosx --show-sdk-path")
	case SDKIos:
		return e.shell.Shell("xcrun --sdk iphoneos --show-sdk-path")
	case SDKIosSimulator:
		return e.shell.Shell("xcrun --sdk iphonesimulator --show-sdk-path")
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
		return e.shell.Shell("xcodebuild -find clang")
	}
	if e.IsAndroidPlatform() {
		return e.getNDKClangPath(false)
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetCXXPath() string {
	return e.cachedString("cxx", e.fetchCXXPath)
}

func (e *OSEnv) fetchCXXPath() string {
	if e.IsDarwinPlatform() {
		return e.shell.Shell("xcodebuild -find clang++")
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
	e.ThrowUnsupportedError()
	panic("unreachable")
}

func (e *OSEnv) GetNDKPath() string {
	if e.IsAndroidPlatform() {
		return e.cachedString("ndk", func() string {
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
		return e.cachedString("ndk_cmake_toolchain", func() string {
			path := filepath.Join(e.GetNDKPath(), "build/cmake/android.toolchain.cmake")
			io2.FileMustExist(path)
			return path
		})
	}
	e.ThrowUnsupportedError()
	panic("unreachable")
}

// Replaces the extension of a library file based on the platform and lib type.
// 'liba.<s>' -> 'liba.a' for static library on all platforms.
// 'liba.<d>' -> 'liba.dylib' or 'liba.so' for dynamic library on different platforms.
func (e *OSEnv) ExpandFilenameLibType(s string) (string, LibType) {
	if strings.HasSuffix(s, LibFilenameSuffixStatic) {
		trimmed := strings.TrimSuffix(s, LibFilenameSuffixStatic)
		libType := LibTypeStatic
		return trimmed + ".a", libType
	}
	if strings.HasSuffix(s, LibFilenameSuffixDynamic) {
		trimmed := strings.TrimSuffix(s, LibFilenameSuffixDynamic)
		libType := LibTypeDynamic
		if e.IsDarwinPlatform() {
			return trimmed + ".dylib", libType
		}
		return trimmed + ".so", libType
	}
	return "", LibTypeStatic
}

func (e *OSEnv) getNDKToolchainRootPath() string {
	ndkPath := e.GetNDKPath()
	return e.cachedString("ndk-toolchain-root", func() string {
		path := filepath.Join(ndkPath, "toolchains", "llvm", "prebuilt", "darwin-x86_64")
		return io2.DirectoryMustExist(path)
	})
}

func (e *OSEnv) GetNDKToolchainBinPath(name string) string {
	return e.cachedString("ndk-toolchain-bin-"+name, func() string {
		path := filepath.Join(e.getNDKToolchainRootPath(), "bin", name)
		return io2.FileMustExist(path)
	})
}

func (e *OSEnv) StripFile(src, dst string) {
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

func (e *OSEnv) AutoVerifyFileArch(baseDir string, outFile []string) {
	if len(outFile) > 0 {
		// Don't update the `outFile` slice in-place.
		outFileCopy := make([]string, len(outFile))
		copy(outFileCopy, outFile)

		// Call `ExpandFilenameLibType` on last element, which is the filename with lib type suffix.
		lastIndex := len(outFileCopy) - 1
		filename, libType := e.ExpandFilenameLibType(outFileCopy[lastIndex])
		if filename == "" {
			e.shell.Quit(fmt.Sprintf("Invalid output filename %s, should end with %s for static library or %s for dynamic library", outFileCopy[lastIndex], LibFilenameSuffixStatic, LibFilenameSuffixDynamic))
		}
		outFileCopy[lastIndex] = filename

		parts := append([]string{baseDir}, outFileCopy...)
		fullPath := filepath.Join(parts...)

		e.VerifyFileArch(libType, fullPath)
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

func (e *OSEnv) checkStaticLibMinSDKVer(file string, minSDKVer string) {
	output := e.shell.Shell(fmt.Sprintf("otool -l %s | grep -m 1 minos", file))
	// output looks like:
	// minos 18.2
	if strings.HasPrefix(output, "minos ") {
		verStr := strings.TrimPrefix(output, "minos ")
		if verStr != minSDKVer {
			e.shell.Quit(fmt.Sprintf("Unexpected min SDK version: %s, expected: %s for file %s", verStr, minSDKVer, file))
		}
		// Success,
		return
	}
	e.shell.Quit(fmt.Sprintf("Cannot find minos in %s", file))
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

func (e *OSEnv) CheckLocalStaticLibMinSDKVer(file string) {
	if e.IsDarwinPlatform() {
		e.checkStaticLibMinSDKVer(file, e.MinDarwinSDKVer())
	}
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
	key = string(e.SDK) + "_" + string(e.Arch) + "_" + key
	return e.stringCache.Get(key, fn)
}
