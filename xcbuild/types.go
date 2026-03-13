package xcbuild

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

type XCContext struct {
	CLIArgs *ku.CLIArgs
	Tunnel  *j9.Tunnel
	Target  string
}

type XCDylibContext struct {
	XCCtx *XCContext
	Info  *XCDylibInfo
	SDK   ku.SDKEnum
}

type XCBuildOptions struct {
	// The fallback target if -target is not provided in CLI args.
	DefaultTarget string

	// The allowed targets for CLI args.
	AllowedTargets []string
	LibNames       []string

	// K: lib name. V: header relative path.
	LibHeaderPathMap map[string]string

	GetModuleMapTargets      func(ctx *XCContext) []string
	GetDylibModuleMapContent func(ctx *XCDylibContext) string

	// An optional subdirectory under lib/ to search for dylibs.
	LibSubDir string

	// Default is false. Only update dependency rpaths that are in the build directory.
	// If true, update all dependency rpaths that are not in /usr/bin.
	AggressiveDepRpathUpdates bool
}

type XCDylibInfo struct {
	// Can be either a lib name or ffapp (ffprobe).
	// Example: libavformat
	Name string
	// Example: libavformat.61.7.100.dylib
	FileName string
}

type iFrameworkInfo struct {
	LibInfo XCDylibInfo
	// Example: sdk-iphoneos/framework/ffprobe/libavformat.framework
	Path string
	// Example: sdk-iphoneos/framework/ffprobe/libavformat.framework/libavformat
	BinPath string
	// Example: sdk-iphoneos/arm64/ffprobe/include/libavformat
	SourceHeadersDir string
	// Example: sdk-iphoneos/arm64/ffprobe/lib/libavformat.dylib
	// Could be multiple if fat.
	SourceDylibPaths []string
	IsFat            bool
}
