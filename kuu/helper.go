package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mgenware/ku-builder"
)

func androidSDKPath(shell *ku.Shell) string {
	// Get $ANDROID_HOME.
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		shell.Quit("$ANDROID_HOME is not set")
		panic("unreachable")
	}
	return androidHome
}

func ndkPath(shell *ku.Shell, ndkVer string) string {
	ndkContainer := filepath.Join(androidSDKPath(shell), "ndk")
	ndkPath := filepath.Join(ndkContainer, ndkVer)
	return ndkPath
}

func ndkBinPath(shell *ku.Shell, ndkVer string, name string) string {
	return filepath.Join(ndkPath(shell, ndkVer), "toolchains", "llvm", "prebuilt", "darwin-x86_64", "bin", name)
}

func mustHaveNDKVer(ndkVer string) string {
	if ndkVer == "" {
		fmt.Println("NDK version not specified. Please use -ndk to specify it.")
		os.Exit(1)
	}
	return ndkVer
}

func printUsage() {
	writeUsage(os.Stdout)
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage: kuu [options] <action> [input]

Run on macOS. Put options before the action.

Actions:
  dep <input>     List binary dependencies (otool -L or llvm-readelf -d).
  symbol <input>  List global symbols (nm -gU or llvm-nm -g).
  deploy          Copy configured artifacts using .ku.json in the current directory.

Options:
  -platform, -p  Platform: macos (m), ios (i), android (a), or darwin (d).
  -target, -t    Deployment target (defaults to deploy_default_target in .ku.json).
  -ndk           Android NDK version under $ANDROID_HOME/ndk.
  -debug, -d     Deploy debug rather than release artifacts.
  -verbose       Show additional action and input information.
  -help, -h      Show this help.

For dep and symbol, .so inputs default to Android; other extensions default to
Darwin. Specify -platform to override detection. Android inspection requires
-ndk and ANDROID_HOME. Supplying -ndk without -platform selects Android.
Deployment requires -platform (or -ndk for Android) and ignores input.

Examples:
  kuu dep ./libexample.dylib
  kuu -platform android -ndk 27.2.12479018 symbol ./libexample.so
  kuu -platform ios -target example deploy`)
}
