package main

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/example"
)

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: "libogg",
	}
	ku.StartLoop(cliOpt, func(ctx *ku.BuildContext) {
		ctx.Tunnel.Logger().Log(j9.LogLevelWarning, "Building target: "+ctx.CLIArgs.Target+" for "+string(ctx.Arch)+" with SDK: "+string(ctx.SDK))

		libInfo := example.BuildOgg(ctx, ku.LibTypeStatic)

		// Go back to the repo root dir.
		ctx.Tunnel.CD(libInfo.RepoDir)
	})
}
