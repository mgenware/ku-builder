package example

import "github.com/mgenware/ku-builder"

var Repo = &ku.SourceRepo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: "libogg",
}

func BuildOgg(ctx *ku.BuildContext, libType ku.LibType) *ku.SourceInfo {
	repoDir := ku.CloneAndGotoRepo(ctx.Tunnel, Repo)

	buildDir := ctx.GetArchBuildDir(string(Repo.Name))
	ctx.Tunnel.CD(buildDir)

	args := ctx.CommonCmakeArgs(libType)
	// repo dir is passed as the last argument.
	args = append(args, repoDir)

	env := ctx.GetCompilerConfigureEnv(nil)
	ctx.RunCmake(&ku.RunCmakeOpt{
		Args: args,
		Env:  env,
	})

	ctx.RunCmakeBuild()
	ctx.RunCmakeInstall()

	libInfo := ku.NewSourceInfo(Repo, repoDir)
	return libInfo
}
