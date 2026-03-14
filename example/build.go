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
	ctx.Shell.CD(buildDir)

	args := ctx.CommonCmakeArgs(libType)
	// repo dir is passed as the last argument.
	args = append(args, repoDir)

	env := ctx.GetCompilerConfigureEnv(nil)
	ctx.RunCmakeGen(&ku.RunCmakeGenOptions{
		Args: args,
		Env:  env,
	})

	ctx.RunCmakeBuild()
	ctx.RunCmakeInstall()

	libInfo := ku.NewSourceInfo(Repo, repoDir)
	return libInfo
}
