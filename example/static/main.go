package main

import (
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/example"
)

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: example.LibName,
	}
	libType := ku.LibTypeStatic
	ku.StartLoop(libType, cliOpt, func(ctx *ku.BuildContext) {
		ctx.LogContext()

		libInfo := example.BuildOgg(ctx)

		// Go back to the repo root dir.
		ctx.Shell.CD(libInfo.RepoDir)
	})
}
