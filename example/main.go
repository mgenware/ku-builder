package main

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

var repo = &ku.SourceRepo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: "libogg",
}

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: "libogg",
	}
	cliArgs := ku.ParseCLIArgs(cliOpt)
	tunnel := ku.CreateDefaultTunnel()

	for _, sdk := range cliArgs.SDKs {
		var archs []ku.ArchEnum
		if cliArgs.Arch != "" {
			archs = append(archs, cliArgs.Arch)
		} else {
			archs = ku.SDKArchs[sdk]
		}

		for _, arch := range archs {
			tunnel.Logger().Log(j9.LogLevelWarning, "Building target: "+cliArgs.Target+" for "+string(arch)+" with SDK: "+string(sdk))

			ctx := ku.NewBuildContext(tunnel, sdk, arch, cliArgs)
			libInfo := buildOgg(ctx)

			// Go back to the repo root dir.
			ctx.Tunnel.CD(libInfo.RepoDir)
		}
	}
}

func buildOgg(ctx *ku.BuildContext) *ku.SourceInfo {
	repoDir := ku.CloneAndGotoRepo(ctx.Tunnel, repo)

	buildDir := ctx.GetArchBuildDir(string(repo.Name))
	ctx.Tunnel.CD(buildDir)

	args := ctx.CommonCmakeArgs()
	// repo dir is passed as the last argument.
	args = append(args, repoDir)

	env := ctx.GetCompilerConfigureEnv(nil)
	ctx.RunCmake(&ku.RunCmakeOpt{
		Args: args,
		Env:  env,
	})

	ctx.RunCmakeBuild()
	ctx.RunCmakeInstall()

	libInfo := ku.NewSourceInfo(repo, repoDir)
	return libInfo
}
