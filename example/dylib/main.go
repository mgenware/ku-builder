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
	loopOpt := &ku.StartLoopOptions{
		LoopFn: func(ctx *ku.BuildContext) {
			ctx.LogContext()

			libInfo := example.BuildOgg(ctx, libType)

			// Go back to the repo root dir.
			ctx.Shell.CD(libInfo.RepoDir)
		},
		AfterAllFn: func(c *ku.CLIArgs, t *j9.Tunnel) {
			ku.CopyJNILibs(c, t, []string{example.LibName + ".so"}, []string{"ogg"})
		},
	}
	ku.StartLoopWithOptions(cliOpt, loopOpt)
}
