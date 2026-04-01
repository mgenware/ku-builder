package main

import (
	"github.com/mgenware/ku-builder"
)

const LibName = "libogg"

var Repo = &ku.RepoInfo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: LibName,
}

func BuildOgg(be *ku.BuildEnv, libType ku.LibType) {
	bp := ku.NewBuildProject(Repo, be, libType)
	bp.CloneAndGotoRepo()
	args := bp.GetCmakeGenArgs()

	env := bp.GetCompilerConfigureEnv(nil)
	bp.RunCmakeGen(&ku.RunCmakeGenOptions{
		Args: args,
		Env:  env,
	})

	bp.GoToBuildDir()
	bp.RunCmakeBuild()
	bp.RunCmakeInstall([]string{"libogg" + libType.ToFilenameSuffix()})
}

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: LibName,
	}
	loopOpt := &ku.StartEnvLoopOptions{
		LoopFn: func(be *ku.BuildEnv) {
			be.LogSummary()

			libType := be.CLIArgs.LibType
			BuildOgg(be, libType)
		},
		AfterAllFn: func(shell *ku.Shell) {
			ku.CopyJNILibs(shell, []string{LibName + ".so"}, []string{"ogg"})
		},
	}
	ku.StartEnvLoopWithOptions(cliOpt, loopOpt)
}
