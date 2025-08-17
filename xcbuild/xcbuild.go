package xcbuild

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/io2"
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
	DefaultTarget  string
	AllowedTargets []string

	GetModuleMapTargets      func(ctx *XCContext) []string
	GetDylibModuleMapContent func(ctx *XCDylibContext) string
}

func Build(opt *XCBuildOptions) {
	if opt == nil {
		fmt.Println("No options provided")
		os.Exit(1)
	}

	cliOpt := &ku.CLIOptions{
		DefaultTarget:   opt.DefaultTarget,
		AllowedTargets:  opt.AllowedTargets,
		DefaultPlatform: ku.PlatformDarwin,
	}
	cliArgs := ku.ParseCLIArgs(cliOpt)
	tunnel := ku.CreateDefaultTunnel()
	buildDir := ku.GetBuildDir(cliArgs.DebugBuild)
	target := cliArgs.Target

	xcCtx := &XCContext{
		CLIArgs: cliArgs,
		Tunnel:  tunnel,
		Target:  target,
	}

	// Target lib names with modulemap (NOTE: key is target lib names).
	moduleMapTargetLibNames := make(map[string]bool)
	if opt.GetModuleMapTargets != nil {
		moduleMapTargets := opt.GetModuleMapTargets(xcCtx)
		for _, target := range moduleMapTargets {
			moduleMapTargetLibNames[ku.GetTargetLibName(target)] = true
		}
	}

	// List of dylib info including ffapp.
	// Initialized in the first SDK loop.
	var dylibInfoList []XCDylibInfo
	sdks := cliArgs.SDKs
	if sdks == nil {
		sdks = ku.PlatformSDKs[ku.PlatformDarwin]
	}
	platformStr := string(cliArgs.PlatformArg)
	if platformStr == "" {
		platformStr = "darwin"
	}

	// K: library name
	// V: framework info
	// Example:
	// K: libavformat
	// V: [
	// 	iphoneos/framework/ffprobe/libavformat.framework,
	// 	iphonesimulator/framework/ffprobe/libavformat.framework
	// ]
	fwMap := make(map[string][]iFrameworkInfo)

	for _, sdk := range sdks {
		sdkDir := ku.GetSDKDir(buildDir, sdk)
		archs := ku.SDKArchs[sdk]
		sdkFwDir := ku.GetSDKFrameworkDir(sdkDir)

		// Get the first arch lib dir to get dylib info.
		firstArchLibDir := ""
		for _, arch := range archs {
			archDir := ku.GetSDKArchDir(sdkDir, arch)
			targetDir := filepath.Join(archDir, target)
			distDir := ku.GetTargetDistDir(targetDir)
			distLibDir := filepath.Join(distDir, "lib")
			if firstArchLibDir == "" {
				firstArchLibDir = distLibDir
			}
		} // end of for archs

		if dylibInfoList == nil {
			dylibInfoList = getDylibInfo(firstArchLibDir)
			fmt.Printf("Found libraries: %v\n", dylibInfoList)
		}

		hasLibModulemapSet := false
		// Used to get headers for the resulting dylib.
		arm64TargetDir := filepath.Join(ku.GetSDKArchDir(sdkDir, ku.ArchArm64), target)

		// Create frameworks.
		for _, dylibInfo := range dylibInfoList {
			// https://developer.apple.com/documentation/bundleresources/placing-content-in-a-bundle
			// iOS and macOS has different framework structures.
			// Use arm64 headers for the resulting dylib.
			dylibCtx := &XCDylibContext{
				Info:  &dylibInfo,
				SDK:   sdk,
				XCCtx: xcCtx,
			}

			arm64DistDir := ku.GetTargetDistDir(arm64TargetDir)
			srcDylibHeadersDir := filepath.Join(arm64DistDir, "include")

			// Check if `../include/${current_dylib}` exists.
			headersWithDylibName := filepath.Join(srcDylibHeadersDir, dylibInfo.Name)
			if io2.DirectoryExists(headersWithDylibName) {
				srcDylibHeadersDir = headersWithDylibName
			} else if !io2.DirectoryExists(srcDylibHeadersDir) {
				fmt.Printf("Headers dir not found: %s\n", srcDylibHeadersDir)
				os.Exit(1)
			}
			isMacos := sdk == ku.SDKMacos
			srcDylibFat := ku.FatSDKs[sdk]
			hasModuleMap := moduleMapTargetLibNames[dylibInfo.Name]

			fwPath := filepath.Join(sdkFwDir, dylibInfo.Name+".framework")
			var fwContentDir string
			if isMacos {
				fwContentDir = filepath.Join(fwPath, "Versions/A")
			} else {
				fwContentDir = fwPath
			}

			var fwInfoPlistDir string
			if isMacos {
				fwInfoPlistDir = filepath.Join(fwContentDir, "Resources")
			} else {
				fwInfoPlistDir = fwContentDir
			}

			fwBinPath := filepath.Join(fwContentDir, dylibInfo.Name)
			fwInfoPlistPath := filepath.Join(fwInfoPlistDir, "Info.plist")
			fwHeadersDir := filepath.Join(fwContentDir, "Headers")
			fwModulesDir := filepath.Join(fwContentDir, "Modules")

			// Clean the framework dir first.
			io2.CleanDir(fwPath)
			io2.Mkdirp(fwInfoPlistDir)
			io2.Mkdirp(fwHeadersDir)
			io2.Mkdirp(fwModulesDir)

			// Loop each arch and create a list of dylib paths.
			var archDylibPaths []string
			for _, arch := range archs {
				archDir := ku.GetSDKArchDir(sdkDir, arch)
				targetDir := filepath.Join(archDir, target)
				distDir := ku.GetTargetDistDir(targetDir)
				archDylibPath := filepath.Join(distDir, "lib", dylibInfo.FileName)
				archDylibPaths = append(archDylibPaths, archDylibPath)
			}

			// Set dylib rpath before lipo.
			for _, archDylibPath := range archDylibPaths {
				// Set dylib rpath.
				tunnel.Spawn(&j9.SpawnOpt{
					Name: "install_name_tool",
					Args: []string{"-id", "@rpath/" + dylibInfo.Name + ".framework/" + dylibInfo.Name, archDylibPath},
				})

				// Set rpath of dependencies.
				updateDylibDepRpath(tunnel, archDylibPath, buildDir)
			}

			// lipo
			var lipoArgs []string
			lipoArgs = append(lipoArgs, "-create")
			// If `archDylibPaths` has multiple items, lipo also creates a fat binary.
			lipoArgs = append(lipoArgs, archDylibPaths...)
			lipoArgs = append(lipoArgs, "-output")
			lipoArgs = append(lipoArgs, fwBinPath)
			tunnel.Spawn(&j9.SpawnOpt{
				Name: "lipo",
				Args: lipoArgs},
			)

			// Add Info.plist
			infoPlistContent := infoPlistForFw(dylibInfo.Name, "com.mgenware", isMacos)
			err := os.WriteFile(fwInfoPlistPath, []byte(infoPlistContent), 0644)
			if err != nil {
				panic(err)
			}

			// Headers
			tunnel.Shell(&j9.ShellOpt{
				Cmd: "cp -R " + filepath.Join(srcDylibHeadersDir, "*") + " " + fwHeadersDir,
			})

			// Modulemap for [target].framework.
			if hasModuleMap {
				fmt.Printf("Creating modulemap for %s\n", dylibInfo.Name)
				fwModuleMapFile := filepath.Join(fwModulesDir, "module.modulemap")

				var fwModuleMapContent string
				if opt.GetDylibModuleMapContent != nil {
					fwModuleMapContent = opt.GetDylibModuleMapContent(dylibCtx)
				} else {
					fwModuleMapContent = moduleMapForFw(dylibInfo.Name)
				}
				err = os.WriteFile(fwModuleMapFile, []byte(fwModuleMapContent), 0644)
				if err != nil {
					panic(err)
				}
				hasLibModulemapSet = true
			}

			// Set up symlinks for macOS framework.
			if isMacos {
				// Create Versions/Current symlink first.
				// Other symlinks depend on this.
				tunnel.Spawn(&j9.SpawnOpt{
					Name: "ln",
					Args: []string{"-s", "A", filepath.Join(fwPath, "Versions/Current")},
				})
				symItems := []string{"Headers", "Resources", dylibInfo.Name}
				if hasModuleMap {
					symItems = append(symItems, "Modules")
				}
				for _, symName := range symItems {
					tunnel.Spawn(&j9.SpawnOpt{
						Name:       "ln",
						Args:       []string{"-s", "Versions/Current/" + symName, filepath.Join(fwPath, symName)},
						WorkingDir: fwPath,
					})
				}
			}

			// Save the dylib path.
			fwInfo := iFrameworkInfo{
				LibInfo:          dylibInfo,
				Path:             fwPath,
				BinPath:          fwBinPath,
				SourceHeadersDir: srcDylibHeadersDir,
				SourceDylibPaths: archDylibPaths,
				IsFat:            srcDylibFat,
			}

			fwMap[dylibInfo.Name] = append(fwMap[dylibInfo.Name], fwInfo)
		} // end of for dylibInfoList
		if !hasLibModulemapSet {
			panic(fmt.Sprintf("No modulemap set for target %s, `moduleMapSet`: %v, `dylibInfoList`: %v", target, moduleMapTargetLibNames, dylibInfoList))
		}
	} // end of for sdks

	xcDir := filepath.Join(buildDir, "xcframework", platformStr)
	io2.CleanDir(xcDir)
	var xcList []string

	// Create xcframeworks.
	for _, dylibInfo := range dylibInfoList {
		xcLibDir := filepath.Join(xcDir, dylibInfo.Name+".xcframework")
		io2.CleanDir(xcLibDir)

		var xcArgs []string
		sdkFwList := fwMap[dylibInfo.Name]
		if len(sdkFwList) == 0 {
			panic("No SDK dylib found for library: " + dylibInfo.Name)
		}

		xcArgs = append(xcArgs, "-create-xcframework")
		for _, sdkFw := range sdkFwList {
			xcArgs = append(xcArgs, "-framework", sdkFw.Path)
		}
		xcArgs = append(xcArgs, "-output")
		xcArgs = append(xcArgs, xcLibDir)
		tunnel.Spawn(&j9.SpawnOpt{
			Name: "xcodebuild",
			Args: xcArgs},
		)
		xcList = append(xcList, xcLibDir)
	}

	// Sign the xcframeworks.
	if !cliArgs.DebugBuild {
		if cliArgs.SignArg == "" {
			panic("-sign is required for release build")
		}
		tunnel.Logger().Log(j9.LogLevelWarning, "Signing xcframeworks")
		for _, xc := range xcList {
			tunnel.Spawn(&j9.SpawnOpt{
				Name: "codesign",
				Args: []string{"--timestamp", "-s", cliArgs.SignArg, xc},
			})
		}
	}
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

func getDylibInfo(libDir string) []XCDylibInfo {
	var builtLibs []XCDylibInfo
	files, err := os.ReadDir(libDir)
	if err != nil {
		panic(fmt.Errorf("failed to read dir: %v during `getLibNames`", err))
	}
	for _, file := range files {
		fileName := file.Name()
		// Skip symbolic links.
		if file.IsDir() || file.Type()&fs.ModeSymlink != 0 {
			continue
		}
		// Skip non dylib files.
		if !strings.HasSuffix(fileName, ".dylib") {
			continue
		}

		libName := strings.Split(fileName, ".")[0]
		builtLib := XCDylibInfo{
			Name:     libName,
			FileName: fileName,
		}

		builtLibs = append(builtLibs, builtLib)
	}
	return builtLibs
}

func infoPlistForFw(libName, org string, isMacos bool) string {
	var platformContent string
	if isMacos {
		platformContent = `<key>LSMinimumSystemVersion</key>
<string>` + ku.MinMacosVersion + `</string>`
	} else {
		platformContent = `<key>MinimumOSVersion</key>
<string>` + ku.MinIosVersion + `</string>`
	}

	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
  <dict>
		<key>CFBundleExecutable</key>
		<string>` + libName + `</string>
    <key>CFBundleIdentifier</key>
    <string>` + org + "." + libName + `</string>
    <key>CFBundleName</key>
    <string>` + libName + `</string>
		<key>CFBundleInfoDictionaryVersion</key>
  	<string>6.0</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
		<key>CFBundlePackageType</key>
  	<string>FMWK</string>
` + platformContent + `
  </dict>
</plist>`
}

func moduleMapForFw(libName string) string {
	return `framework module ` + libName + ` {
	header "` + libName + `.h"

	export *
}`
}

func updateDylibDepRpath(t *j9.Tunnel, dylibPath string, buildDir string) {
	output := t.Shell(&j9.ShellOpt{
		Cmd: "otool -L \"" + dylibPath + "\"",
	})
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	// Skip the first line, which is the dylib path.
	lines = lines[1:]

	var keys []string
	var values []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, buildDir) {
			continue
		}
		// Example: <build>/sdk-macosx/arm64/ffprobe/lib/libswresample.5.dylib (compatibility version 5.0.0, current version 5.3.100)
		parenthesesIndex := strings.Index(line, "(")
		var resultLine string
		if parenthesesIndex == -1 {
			resultLine = line
		} else {
			resultLine = strings.TrimSpace(line[:parenthesesIndex])
		}
		keys = append(keys, resultLine)

		// Extract the dylib name without version and extension.
		fileName := filepath.Base(resultLine)
		fileName = strings.Split(fileName, ".")[0]
		values = append(values, fileName)
	}

	// Set dep rpath.
	if len(keys) > 0 {
		var args []string
		for i := 0; i < len(keys); i++ {
			args = append(args, "-change", keys[i], "@rpath/"+values[i]+".framework/"+values[i])
		}
		args = append(args, dylibPath)
		t.Spawn(&j9.SpawnOpt{
			Name: "install_name_tool",
			Args: args,
		})
	}
}
