package main

import (
	"fmt"
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
	fmt.Println("Usage: kuu [options] <action> <input>")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  dep       List dependencies of the input file")
	fmt.Println("  symbol    List exported symbols of the input file")
	fmt.Println("  deploy    Run deployment for the specified target and platform. Input is ignored.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -platform  Platform. Supported platforms: macos(m), ios(i), android(a), darwin(d).")
	fmt.Println("  -p         -platform shorthand.")
	fmt.Println("  -target    Build target.")
	fmt.Println("  -t         -target shorthand.")
	fmt.Println("  -ndk       NDK version.")
	fmt.Println("  -debug     Debug build.")
	fmt.Println("  -help      Show usage information.")
}
