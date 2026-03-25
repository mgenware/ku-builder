package example

import "github.com/mgenware/ku-builder"

const LibName = "libogg"

var Repo = &ku.SourceRepo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: LibName,
}

func BuildOgg(ctx *ku.BuildContext, libType ku.LibType) *ku.SourceInfo {
	repoDir := ku.CloneAndGotoRepo(ctx.Shell, Repo)

	buildDir := ctx.GetArchBuildDir(Repo.Name)
	args := ctx.GetCmakeGenArgs(libType, buildDir)

	env := ctx.GetCompilerConfigureEnv(nil)
	ctx.RunCmakeGen(&ku.RunCmakeGenOptions{
		Args: args,
		Env:  env,
	})

	ctx.Shell.CD(buildDir)
	ctx.RunCmakeBuild()
	ctx.RunCmakeInstall([]string{"libogg" + libType.ToFilenameSuffix()})

	libInfo := ku.NewSourceInfo(Repo, repoDir)
	return libInfo
}
