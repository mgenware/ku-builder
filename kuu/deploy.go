package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

func RunKuDeploy(shell *ku.Shell, target string, debug bool, platform ku.PlatformEnum) {
	rootKuConfig := ReadKuConfig(shell)

	if platform == "" {
		shell.Quit("No platform specified")
	}
	platformStr := string(platform)

	defaultTarget := ReadConfigString(rootKuConfig, "deploy_default_target")
	if target == "" {
		if defaultTarget == "" {
			shell.Quit("No target specified and no default target set in config.")
		}
		target = defaultTarget
	}

	targetConfigMap := ReadConfigMap(rootKuConfig, "deploy_targets")
	if len(targetConfigMap) <= 0 {
		shell.Quit("No target config map found in config.")
	}

	targetConfig := ReadConfigMap(targetConfigMap, target)
	if len(targetConfig) <= 0 {
		shell.Quit(fmt.Sprintf("No config found for target: %s", target))
	}

	srcNames := ReadConfigStringArray(targetConfig, "src_names")
	darwinDestDir := resolveUserDir(ReadConfigString(targetConfig, "dest_dir_darwin"))
	androidDestDir := resolveUserDir(ReadConfigString(targetConfig, "dest_dir_android"))

	buildTypeDir := ku.GetBuildTypeDir(debug)

	if debug {
		shell.Log(j9.LogLevelWarning, "☢️ You are deploying a debug build.")
	}

	fmt.Printf("--- Target config: %s ---\n%v\n--- --- --- --- ---\n", target, targetConfig)

	switch platform {
	case ku.PlatformAndroid:
		shell.Log(j9.LogLevelInfo, fmt.Sprintf("🚕 Deploying to Android: %s", androidDestDir))

		if len(androidDestDir) <= 0 {
			shell.Quit("No Android destination directory specified in config.")
		}

		ku.CopyJNILibsCore(&ku.CopyJNILibsOptions{
			Shell:        shell,
			DstLibsDir:   androidDestDir,
			LibFileNames: srcNames,
			Target:       target,
			Debug:        debug,
			KuDeploy:     true,
		})

	case ku.PlatformDarwin, ku.PlatformIos, ku.PlatformMacos:
		shell.Log(j9.LogLevelVerbose, fmt.Sprintf("🚕 Deploying to %s: %s", platformStr, darwinDestDir))

		if len(darwinDestDir) <= 0 {
			shell.Quit(fmt.Sprintf("No %s destination directory specified in config.", platformStr))
		}

		xcRootDir := ku.GetXCFrameworkDir(buildTypeDir)
		xcDir := filepath.Join(xcRootDir, platformStr, target)
		deployDarwin(shell, xcDir, srcNames, darwinDestDir)

	default:
		shell.Quit(fmt.Sprintf("Unsupported platform: %s", platform))

	}
}

func deployDarwin(shell *ku.Shell, xcDir string, srcNames []string, darwinDestDir string) {
	for _, srcName := range srcNames {
		srcFileName := srcName + ".xcframework"
		src := filepath.Join(xcDir, srcFileName)

		ku.CPToDirByForce(shell, src, true, darwinDestDir)
		shell.Log(j9.LogLevelInfo, fmt.Sprintf("✅ Deployed %s to %s", srcFileName, darwinDestDir))
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
