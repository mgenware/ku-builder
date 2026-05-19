package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/ku-builder"
)

func RunKuDeploy(shell *ku.Shell, target string, debug bool, platform ku.PlatformEnum) {
	InitKuConfig(shell)

	defaultTarget := ReadKuConfigString("deploy_default_target")
	srcNames := ReadKuConfigStringArray("deploy_src_names")
	darwinDestDir := resolveUserDir(ReadKuConfigString("deploy_dest_dir_darwin"))
	androidDestDir := resolveUserDir(ReadKuConfigString("deploy_dest_dir_android"))

	buildTypeDir := ku.GetBuildTypeDir(debug)
	if target == "" {
		if defaultTarget == "" {
			shell.Quit("No target specified and no default target set in config.")
		}
		target = defaultTarget
	}

	if platform == "" {
		shell.Quit("No platform specified")
	}
	platformStr := string(platform)

	if platform == ku.PlatformAndroid {
		ku.CopyJNILibsCore(&ku.CopyJNILibsOptions{
			Shell:        shell,
			DstLibsDir:   androidDestDir,
			LibFileNames: srcNames,
			Target:       target,
			Debug:        debug,
			KuDeploy:     true,
		})
		return
	}

	xcRootDir := ku.GetXCFrameworkDir(buildTypeDir)
	xcDir := filepath.Join(xcRootDir, platformStr, target)
	deployDarwin(shell, xcDir, srcNames, darwinDestDir)
}

func deployDarwin(shell *ku.Shell, xcDir string, srcNames []string, darwinDestDir string) {
	for _, srcName := range srcNames {
		srcFileName := srcName + ".xcframework"
		src := filepath.Join(xcDir, srcFileName)

		ku.CPToDirByForce(shell, src, true, darwinDestDir)
		fmt.Printf("✅ Deployed %s to %s\n", srcFileName, darwinDestDir)
	}
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
