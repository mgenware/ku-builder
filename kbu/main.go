package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

func main() {
	ndkPtr := flag.String("ndk", "", "Specify NDK version.")

	var platformInput string
	var resolvedPlatform ku.PlatformEnum
	flag.StringVar(&platformInput, "platform", "", "Platform. Supported platforms: macos(m), ios(i), android(a), darwin(d).")
	flag.StringVar(&platformInput, "p", "", "-platform shorthand.")

	var target string
	flag.StringVar(&target, "target", "", "Build target.")
	flag.StringVar(&target, "t", "", "-target shorthand.")

	debugPtr := flag.Bool("debug", false, "Debug build.")
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

	if len(args) < 1 {
		printUsage()
		return
	}

	ndkVer := *ndkPtr
	resolvedPlatform = ku.ParsePlatformString(platformInput, false)
	action := args[0]
	var input string
	if len(args) > 1 {
		input = args[1]
	}
	if ndkVer != "" && resolvedPlatform == "" {
		resolvedPlatform = ku.PlatformAndroid
	}

	t := j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())
	shell := ku.NewShell(t, nil)

	switch action {
	case "dep":
		isDarwin, err := getIsDarwinFromInput(input, resolvedPlatform)
		if err != nil {
			shell.Quit(fmt.Sprintf("Error: %v\n", err))
		}
		if isDarwin {
			t.Spawn(&j9.SpawnOpt{Name: "otool", Args: []string{"-L", input}})
		} else {
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath(shell, mustHaveNDKVer(ndkVer), "llvm-readelf"), Args: []string{"-d", input}})
		}

	case "symbol":
		isDarwin, err := getIsDarwinFromInput(input, resolvedPlatform)
		if err != nil {
			shell.Quit(fmt.Sprintf("Error: %v\n", err))
		}
		if isDarwin {
			t.Spawn(&j9.SpawnOpt{Name: "nm", Args: []string{"-gU", input}})
		} else {
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath(shell, mustHaveNDKVer(ndkVer), "llvm-nm"), Args: []string{"-g", input}})
		}

	case "deploy":
		RunKuDeploy(shell, target, *debugPtr, resolvedPlatform)

	default:
		shell.Quit("Unknown action")
	}
}

func getIsDarwinFromInput(input string, platform ku.PlatformEnum) (bool, error) {
	if input == "" {
		return false, fmt.Errorf("no input provided")
	}

	if platform != "" {
		return platform == ku.PlatformDarwin || platform == ku.PlatformIos || platform == ku.PlatformMacos, nil
	}

	// Now we have to guess if it's a Darwin or Android binary based on the file extension.
	var isDarwin bool
	inputExt := filepath.Ext(input)
	if inputExt == ".so" {
		isDarwin = false
	} else {
		isDarwin = true
	}
	return isDarwin, nil
}
