package main

import (
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/example/png"
	"github.com/mgenware/ku-builder/example/zlib"
)

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: "libpng",
	}
	loopOpt := &ku.StartEnvLoopOptions{
		LoopFn: func(be *ku.BuildEnv) {
			be.LogSummary()

			zlib.BuildZlib(be)
			png.BuildPng(be)
		},
	}
	ku.StartEnvLoopWithOptions(cliOpt, loopOpt)
}
