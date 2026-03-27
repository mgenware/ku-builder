package main

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/example"
)

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: example.LibName,
	}
	libType := ku.LibTypeDynamic
	loopOpt := &ku.StartEnvLoopOptions{
		LoopFn: func(be *ku.BuildEnv) {
			be.LogSummary()
			example.BuildOgg(be, libType)
		},
		AfterAllFn: func(c *ku.CLIArgs, t *j9.Tunnel) {
			ku.CopyJNILibs(c, t, []string{example.LibName + ".so"}, []string{"ogg"})
		},
	}
	ku.StartEnvLoopWithOptions(cliOpt, loopOpt)
}
