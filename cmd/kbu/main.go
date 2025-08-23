package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

const configDirName = ".ku-builder"
const ndkConfigFileName = "ndk-ver"

func main() {
	args := os.Args[1:]

	// This program only runs on macOS.
	if runtime.GOOS != "darwin" {
		fmt.Println("This program only runs on macOS.")
		return
	}

	if len(args) < 2 {
		fmt.Println("Usage: kbu <action> <input>")
		return
	}

	action := args[0]
	input := args[1]

	if input == "" {
		fmt.Println("No input provided")
		return
	}

	var isDarwin bool
	inputExt := filepath.Ext(input)
	if inputExt == ".dylib" {
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
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath("llvm-readelf"), Args: []string{"-d", input}})
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

func ndkPath() string {
	ndkContainer := filepath.Join(androidSDKPath(), "ndk")
	ndkVer := getNDKVersion()
	ndkPath := filepath.Join(ndkContainer, ndkVer)
	return ndkPath
}

func ndkBinPath(name string) string {
	return filepath.Join(ndkPath(), "toolchains", "llvm", "prebuilt", "darwin-x86_64", "bin", name)
}

func getNDKVersion() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	ndkConfigFile := filepath.Join(homeDir, configDirName, ndkConfigFileName)
	if io2.FileExists(ndkConfigFile) {
		ndkName, err := os.ReadFile(ndkConfigFile)
		if err != nil {
			panic(err)
		}
		return strings.TrimSpace(string(ndkName))
	}
	panic("NDK version not configured. Please use `setndk` to configure it.")
}
