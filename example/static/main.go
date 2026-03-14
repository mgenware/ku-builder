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
	ku.StartLoop(cliOpt, func(ctx *ku.BuildContext) {
		ctx.Shell.Logger().Log(j9.LogLevelWarning, "Building target: "+ctx.CLIArgs.Target+" for "+string(ctx.Arch)+" with SDK: "+string(ctx.SDK))

		libInfo := example.BuildOgg(ctx, ku.LibTypeStatic)

		// Go back to the repo root dir.
		ctx.Shell.CD(libInfo.RepoDir)
	})
}
