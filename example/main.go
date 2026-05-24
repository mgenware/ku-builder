package main

import (
	"github.com/mgenware/ku-builder"
)

const kTarget = "libogg"

var Repo = &ku.RepoInfo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.6",
	Name: kTarget,
}

func BuildOgg(be *ku.BuildEnv, libType ku.LibType) {
	p := ku.NewCMakeProject(Repo, be, libType)
	p.Init(nil)
	p.Build()
	p.Install([]string{kTarget + libType.ToFilenameSuffix()})
}

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: kTarget,
	}
	loopOpt := &ku.StartEnvLoopOptions{
		LoopFn: func(be *ku.BuildEnv) {
			be.LogSummary()

			libType := be.CLIArgs.LibType
			BuildOgg(be, libType)
		},
	}
	ku.StartEnvLoopWithOptions(cliOpt, loopOpt)
}
