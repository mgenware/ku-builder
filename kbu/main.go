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

	t := j9.NewTunnel(j9.NewLocalNode(), j9.NewConsoleLogger())
	shell := ku.NewShell(t, nil)

	switch action {
	case "dep":
		isDarwin, err := getIsDarwinFromInput(input, osPtr)
		if err != nil {
			shell.Quit(fmt.Sprintf("Error: %v\n", err))
		}
		if isDarwin {
			t.Spawn(&j9.SpawnOpt{Name: "otool", Args: []string{"-L", input}})
		} else {
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath(shell, mustHaveNDKVer(ndkVer), "llvm-readelf"), Args: []string{"-d", input}})
		}

	case "symbol":
		isDarwin, err := getIsDarwinFromInput(input, osPtr)
		if err != nil {
			shell.Quit(fmt.Sprintf("Error: %v\n", err))
		}
		if isDarwin {
			t.Spawn(&j9.SpawnOpt{Name: "nm", Args: []string{"-gU", input}})
		} else {
			t.Spawn(&j9.SpawnOpt{Name: ndkBinPath(shell, mustHaveNDKVer(ndkVer), "llvm-nm"), Args: []string{"-g", input}})
		}

	case "deploy":
		RunKuDeploy(shell)

	default:
		shell.Quit("Unknown action")
	}
}

func getIsDarwinFromInput(input string, osPtr *string) (bool, error) {
	if input == "" {
		return false, fmt.Errorf("no input provided")
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
			return false, fmt.Errorf("invalid OS type. Please specify 'd' for Darwin or 'a' for Android")
		}
	} else if inputExt == ".so" {
		isDarwin = false
	} else {
		isDarwin = true
	}
	return isDarwin, nil
}
