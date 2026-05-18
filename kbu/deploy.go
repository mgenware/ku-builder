package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/ku-builder"
)

func Deploy() {
	InitKuConfig()

	defaultTarget := ReadKuConfigString("deploy_default_target")
	srcNames := ReadKuConfigStringArray("deploy_src_names")
	darwinDestDir := resolveUserDir(ReadKuConfigString("deploy_dest_dir_darwin"))
	// androidDestDir := resolveUserDir(ReadKuConfigString("deploy_dest_dir_android"))

	cliOpt := &ku.CLIOptions{
		DefaultTarget:   defaultTarget,
		DefaultPlatform: ku.PlatformDarwin,
	}
	cliArgs := ku.ParseCLIArgs(cliOpt)

	buildTypeDir := ku.GetBuildTypeDir(cliArgs.DebugBuild)
	target := cliArgs.Target

	if cliArgs.PlatformArg == "" {
		fmt.Printf("No platform specified")
		return
	}

	platformStr := string(cliArgs.PlatformArg)
	if cliArgs.PlatformArg == ku.PlatformAndroid {
		fmt.Printf("Android deployment is not implemented yet")
		return
	}

	xcRootDir := ku.GetXCFrameworkDir(buildTypeDir)
	xcDir := filepath.Join(xcRootDir, platformStr, target)

	for _, srcName := range srcNames {
		srcFileName := srcName + ".xcframework"
		src := filepath.Join(xcDir, srcFileName)
		dest := filepath.Join(darwinDestDir, srcFileName)
		err := copyPath(src, dest, true)
		if err != nil {
			fmt.Printf("Failed to deploy %s: %v\n", srcFileName, err)
		} else {
			fmt.Printf("Deployed %s to %s\n", srcFileName, dest)
		}
	}
}

func copyPath(src, dest string, isDir bool) error {
	// Delete dest path if it exists.
	if _, err := os.Stat(dest); err == nil {
		if isDir {
			err = os.RemoveAll(dest)
		} else {
			err = os.Remove(dest)
		}
		if err != nil {
			return fmt.Errorf("failed to delete existing destination: %v", err)
		}
	}

	destParent := filepath.Dir(dest)
	if err := os.MkdirAll(destParent, 0755); err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}

	if isDir {
		return os.CopyFS(dest, os.DirFS(src))
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

func resolveUserDir(dir string) string {
	if strings.Contains(dir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return dir
		}
		return strings.Replace(dir, "~", homeDir, 1)
	}
	return dir
}
