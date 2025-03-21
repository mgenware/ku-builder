package ku

import (
	"path/filepath"
)

const MinMacosVersion = "11.0"
const MinIosVersion = "14.0"
const MinAndroidAPI = "28"

var ProjectRepoDir string
var buildDir string

type PlatformEnum string

const (
	PlatformMacos   PlatformEnum = "macos"
	PlatformIos     PlatformEnum = "ios"
	PlatformDarwin  PlatformEnum = "darwin"
	PlatformAndroid PlatformEnum = "android"
)

var SupportedPlatforms = map[PlatformEnum]bool{
	PlatformMacos:   true,
	PlatformIos:     true,
	PlatformDarwin:  true,
	PlatformAndroid: true,
}

type ArchEnum string

const (
	ArchArm64  ArchEnum = "arm64"
	ArchX86_64 ArchEnum = "x86_64"
)

var SupportedArchs = map[ArchEnum]bool{
	ArchArm64:  true,
	ArchX86_64: true,
}

type SDKEnum string

const (
	SDKMacos        SDKEnum = "macosx"
	SDKIos          SDKEnum = "iphoneos"
	SDKIosSimulator SDKEnum = "iphonesimulator"
	SDKAndroid      SDKEnum = "android"
)

var SupportedSDKs = map[SDKEnum]bool{
	SDKMacos:        true,
	SDKIos:          true,
	SDKIosSimulator: true,
	SDKAndroid:      true,
}

// SDKs that need to be built as fat binaries.
var FatSDKs = map[SDKEnum]bool{
	SDKMacos:        true,
	SDKIosSimulator: true,
}

var PlatformSDKs = map[PlatformEnum][]SDKEnum{
	PlatformMacos:   {SDKMacos},
	PlatformIos:     {SDKIos, SDKIosSimulator},
	PlatformDarwin:  {SDKMacos, SDKIos, SDKIosSimulator},
	PlatformAndroid: {SDKAndroid},
}

var SDKArchs = map[SDKEnum][]ArchEnum{
	SDKMacos:        {ArchArm64, ArchX86_64},
	SDKIos:          {ArchArm64},
	SDKIosSimulator: {ArchArm64},
	SDKAndroid:      {ArchArm64, ArchX86_64},
}

func GetBuildDir(debug bool) string {
	if debug {
		return filepath.Join(buildDir, "debug")
	}
	return filepath.Join(buildDir, "release")
}

func GetSDKDir(buildDir string, sdk SDKEnum) string {
	return filepath.Join(buildDir, "sdk-"+string(sdk))
}

func GetSDKArchDir(sdkDir string, arch ArchEnum) string {
	return filepath.Join(sdkDir, string(arch))
}

func GetSDKFrameworkDir(sdkDir string) string {
	return filepath.Join(sdkDir, "framework")
}

func GetSDKXCFrameworkDir(sdkDir string) string {
	return filepath.Join(sdkDir, "xcframework")
}

func GetOldArch(arch ArchEnum) string {
	if arch == ArchArm64 {
		return "aarch64"
	}
	return "x86_64"
}

func GetABI(arch ArchEnum) string {
	if arch == ArchArm64 {
		return "arm64-v8a"
	}
	return "x86_64"
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}

func init() {
	ProjectRepoDir = mustAbs("./repo")
	buildDir = mustAbs("./build")
}
