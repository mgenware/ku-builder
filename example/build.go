package example

import "github.com/mgenware/ku-builder"

const LibName = "libogg"

var Repo = &ku.RepoInfo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: LibName,
}

func BuildOgg(env *ku.BuildEnv, libType ku.LibType) *ku.SourceInfo {
	repoDir := ku.CloneAndGotoRepo(env.Shell, Repo)

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
