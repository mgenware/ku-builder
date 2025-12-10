package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mgenware/j9/v3"
)

func main() {
	ndkPtr := flag.String("ndk", "", "Specify NDK version.")
	osPtr := flag.String("os", "", "d (Darwin) or a (Android). If not specified. Detect from input file.")
	helpPtr := flag.Bool("help", false, "Show usage information.")
	flag.Parse()
	args := flag.Args()

	// This program only runs on macOS.
	if runtime.GOOS != "darwin" {
		fmt.Println("This program only runs on macOS.")
		return
	}

	if *helpPtr {
		printUsage()
		return
	}

	if len(args) < 2 {
		printUsage()
		return
	}

	ndkVer := *ndkPtr
	action := args[0]
	input := args[1]

	if input == "" {
		fmt.Println("No input provided")
		return
	}

	var isDarwin bool
	inputExt := filepath.Ext(input)
	if *osPtr != "" {
		v := *osPtr
		switch v {
		case "d":
			isDarwin = true
		case "a":
			isDarwin = false
		default:
			fmt.Println("Invalid OS type. Please specify 'd' for Darwin or 'a' for Android.")
			return
		}
	} else if inputExt == ".dylib" {
		isDarwin = true
	} else if inputExt != ".so" {
		fmt.Println("Unsupported file type")
		return
	}

	t := j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())

	switch action {
	case "deps":
		if isDarwin {
			t.Spawn(&j9.SpawnOpt{Name: "otool", Args: []string{"-L", input}})
		} else {
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath(mustHaveNDKVer(ndkVer), "llvm-readelf"), Args: []string{"-d", input}})
		}

	default:
		fmt.Println("Unknown action")
	}
}

func androidSDKPath() string {
	// Get $ANDROID_HOME.
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		panic("$ANDROID_HOME is not set")
	}
	return androidHome
}

func ndkPath(ndkVer string) string {
	ndkContainer := filepath.Join(androidSDKPath(), "ndk")
	ndkPath := filepath.Join(ndkContainer, ndkVer)
	return ndkPath
}

func ndkBinPath(ndkVer string, name string) string {
	return filepath.Join(ndkPath(ndkVer), "toolchains", "llvm", "prebuilt", "darwin-x86_64", "bin", name)
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
	fmt.Println("Actions:")
	fmt.Println("  deps       List dependencies of the input file")
	fmt.Println("Options:")
	fmt.Println("  -ndk       Specify NDK version.")
	fmt.Println("  -os        Specify the operating system type: 'd' for Darwin, 'a' for Android")
}
