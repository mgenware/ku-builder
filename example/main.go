package main

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

var repo = &ku.SourceRepo{
	Url:  "https://chromium.googlesource.com/webm/libwebp",
	Name: "libwebp",
	Tag:  "v1.5.0",
}

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: repo.Name,
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

			opt := ku.NewBuildContextInitOpt(tunnel, sdk, arch, cliArgs)
			ctx := ku.NewBuildContext(opt)
			libInfo := buildLibwebp(ctx)

			// Go back to the repo root dir.
			ctx.Tunnel.CD(libInfo.RepoDir)
		}
	}
}

func buildLibwebp(ctx *ku.BuildContext) *ku.SourceInfo {
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
