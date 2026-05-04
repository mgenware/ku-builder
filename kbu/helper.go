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
	fmt.Println("Usage: kbu <action> [options] <input>")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  deps       List dependencies of the input file")
	fmt.Println("  symbols    List exported symbols of the input file")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -ndk       Specify NDK version")
	fmt.Println("  -os        Specify the operating system type: 'd' for Darwin, 'a' for Android")
}
