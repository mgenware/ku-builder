package xcbuild

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mgenware/ku-builder"
)

func Deploy(opt *XCBuildOptions) {
	if opt == nil {
		fmt.Println("No options provided")
		os.Exit(1)
	}

	cliOpt := &ku.CLIOptions{
		DefaultTarget:   opt.DefaultTarget,
		AllowedTargets:  opt.AllowedTargets,
		DefaultPlatform: ku.PlatformDarwin,
	}
	cliArgs := ku.ParseCLIArgs(cliOpt)

	buildTypeDir := ku.GetBuildTypeDir(cliArgs.DebugBuild)
	target := cliArgs.Target
	platformStr := string(cliArgs.PlatformArg)
	if platformStr == "" {
		platformStr = "darwin"
	}
	xcRootDir := ku.GetXCFrameworkDir(buildTypeDir)
	xcDir := filepath.Join(xcRootDir, platformStr, target)
}
