# ku-builder

ku-builder is a Go library for building native libraries with CMake, Make, or
Meson for Apple and Android targets from macOS. It provides source checkout,
toolchain configuration, per-SDK build directories, and helpers for packaging
and deploying artifacts. It is not a general-purpose `ku-builder` executable:
write a Go program using the library, or use the included examples.

## Requirements and targets

- Host: latest stable macOS; Go 1.25 or newer.
- Apple builds: Xcode and the relevant SDKs, available through `xcodebuild` and
  `xcrun`. XCFramework packaging also uses Apple command-line tools.
- Install the build system required by the project (`cmake`, `make`, or
  `meson` and its backend), plus `pkg-config` and Git. Source downloads need
  network access unless the sources are already available locally.
- Android builds: an installed Android NDK. The builder resolves `-ndk` as an
  absolute NDK path or as a version under `$ANDROID_SDK_PATH/ndk`; when
  `ANDROID_SDK_PATH` is unset, it uses `~/Library/Android/sdk`. An
  `ANDROID_NDK_PATH` environment variable overrides the resolved path, but
  `-ndk` is still required by the argument parser.

| Platform | SDK (`-sdk`) | Architectures | Minimum OS/API |
| --- | --- | --- | --- |
| macOS | `macosx` | `arm64`, `x86_64` | macOS 11 |
| iOS device | `iphoneos` | `arm64` | iOS 14 |
| iOS Simulator | `iphonesimulator` | `arm64` | iOS 14 |
| Android | `android` | `arm64`, `x86_64` | API 26 |

## Build example

The [example](example/main.go) builds zlib followed by libpng using CMake.
It installs both into the same target output directory so libpng can find zlib.
Run it from the `example` directory: source and build paths are based on the
process's **initial working directory**, not the location of the Go source.

```bash
cd example
go run . -help
go run . -platform macos -arch arm64
go run . -sdk iphoneos
go run . -platform android -ndk 27.2.12479018
```

Replace the example NDK version with one installed on your machine. The first
build can clone sources into `example/repo`; existing Git checkouts may be
pulled unless `-no-pull` is set. The example defaults to target `libpng`, builds
static libraries, and always runs its `Init`, `Build`, and `Install` calls.

Each SDK/architecture iteration writes to:

```text
build/{release|debug}/sdk-<sdk>/<arch>/<target>/
  out/                  installed headers and libraries
  tmp/build/<repo>/     temporary build files
  dist/                 optional distribution files, if enabled by the program
```

**The environment loop cleans each target's `out` directory before running** by
default. `-clean` additionally cleans temporary build directories when they are
created. Programs can set `StartEnvLoopOptions.DisableAutoClean` to preserve
`out`; take care not to run a build against directories containing files you
need to keep.

## Build-program CLI

Programs using `ku.ParseCLIArgs` or `ku.StartEnvLoopWithOptions` get these flags.
Use `-help` to see the defaults and targets configured by that particular
program. Go's flag parser stops at the first positional argument, so put flags
before positional arguments.

| Flag | Description |
| --- | --- |
| `-platform`, `-p` | `macos` (`m`), `ios` (`i`, device and simulator), `android` (`a`), or `darwin` (`d`, macOS and iOS). |
| `-sdk` | Select one SDK from the target table; overrides the platform's SDK selection. |
| `-arch` | `arm64` or `x86_64`; omitted means all supported architectures per SDK. An explicitly selected architecture must be supported by every selected SDK. |
| `-target`, `-t` | Target name, restricted by the calling program's `CLIOptions`. A configured default may make it optional. |
| `-ndk` | Android NDK version or absolute path; required when the selected SDK is Android. With no platform, supplying it defaults the platform to Android. |
| `-debug`, `-d` | Use debug instead of release settings and output directories. |
| `-clean` | Clean temporary build directories before configuring. |
| `-no-pull` | Skip `git pull` for existing source checkouts; does not prevent an initial clone. |
| `-action` | `configure`, `clean`, `build`, or `make`; the calling program must interpret this value. The parser does not run an action itself. |
| `-dylib` | Request dynamic rather than static libraries through `CLIArgs.LibType`; the calling program must use that value. |
| `-sign` | Signing identity for programs that support signing. |
| `-hardened` | Request hardened runtime for macOS framework signing. |

For example, `example/main.go` explicitly passes `LibTypeStatic` and does not
dispatch on `-action`, so neither flag changes its build sequence or library
type. Program-specific defaults and callbacks are defined in `CLIOptions`.

## Writing a build program

1. Define a `RepoInfo` with a repository URL, name and tag, commit, or branch;
   alternatively set `LocalRepoDir` to an existing source directory.
2. Set `CLIOptions` with a default target or an allowed target list. Use
   `StartEnvLoopWithOptions` to run a callback for each selected SDK and
   architecture.
3. In the callback, create a `NewCMakeProject`, `NewMakeProject`, or
   `NewMesonProject`, then call `Init`, `Build`, and `Install`. Use
   `ProjectInitOptions` for project-specific setup arguments and environment
   variables. See [zlib.go](example/zlib/zlib.go) and
   [png.go](example/png/png.go) for a dependency and consumer.
4. `Install("libname", nil)` verifies the installed library's architecture and,
   for Apple static libraries, minimum SDK version. For dynamic libraries,
   pass appropriate `VerifyFileOptions`.

`CLIArgs`, `BuildEnv`, and `OSEnv` expose the selected options, output paths,
and toolchain information. For custom build steps, use `Project.CoreBuilder()`
and the builder helpers. `CLIOptions.CreateDistDir` creates `dist/include` and
`dist/lib`, but the program must populate them.

The [xcbuild](xcbuild/xcbuild.go) package packages existing dynamic libraries
and headers from each target's `dist` directories into Apple frameworks and
XCFrameworks. [example/xcbuild](example/xcbuild/main.go) demonstrates its
configuration for `libogg`; it requires separately built `libogg` dist
artifacts and does not package the zlib/libpng build example automatically.
Run packaging from the same initial working directory used for the builds.

## ku-builder Utils CLI (`kuu`)

`kuu` is a separate macOS utility for inspecting binaries and deploying
previously built artifacts.

```bash
go install github.com/mgenware/ku-builder/kuu@latest
kuu -help
kuu dep ./libexample.dylib
kuu -platform android -ndk 27.2.12479018 symbol ./libexample.so
kuu -platform ios -target example deploy
```

Syntax: `kuu [options] <action> [input]`. Put all options **before** the action.

- `dep <input>` lists dependencies using `otool -L` on Apple binaries or
  `llvm-readelf -d` on Android binaries.
- `symbol <input>` lists global symbols using `nm -gU` or `llvm-nm -g`.
- `deploy` copies configured artifacts; any input argument is ignored.

Options are `-platform`/`-p`, `-target`/`-t`, `-ndk`, `-debug`/`-d`,
`-verbose`, and `-help`/`-h`. For inspection, `.so` defaults to Android and
other extensions to Darwin; use `-platform` to override detection. Android
inspection needs `-ndk <version>` and `ANDROID_HOME`, and looks for tools under
`$ANDROID_HOME/ndk/<version>` (unlike the build-program NDK path settings).
Supplying `-ndk` without a platform selects Android.

### Deployment configuration

Run `kuu deploy` from the directory containing `.ku.json` and the `build`
directory. A platform is required; `-target` can be omitted if
`deploy_default_target` is set. For example:

```json
{
  "deploy_default_target": "example",
  "deploy_targets": {
    "example": {
      "src_names": ["libexample"],
      "dest_dir_darwin": "~/MyApp/Frameworks",
      "dest_dir_android": "~/MyApp/app/src/main/jniLibs"
    }
  }
}
```

For Apple platforms, each name is read as `<name>.xcframework` under
`build/{release|debug}/xcframework/<platform>/<target>/`. For Android, each
name is read as `<name>.so` (or as-is if already ending in `.so`) from each
architecture's `dist/lib`, then copied to `arm64-v8a` and `x86_64` under the
destination. Both Android architectures must have been built. `-debug` selects
debug artifacts; otherwise release artifacts are used. Deployment overwrites
existing destination artifacts.
