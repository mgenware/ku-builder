package example

import "github.com/mgenware/ku-builder"

const LibName = "libogg"

var Repo = &ku.RepoInfo{
	Url:  "https://github.com/xiph/ogg",
	Tag:  "v1.3.5",
	Name: LibName,
}

func BuildOgg(be *ku.BuildEnv, libType ku.LibType) {
	bp := ku.NewBuildProject(Repo, be, libType)
	bp.CloneAndGotoRepo()
	args := bp.GetCmakeGenArgs()

	env := bp.GetCompilerConfigureEnv(nil)
	bp.RunCmakeGen(&ku.RunCmakeGenOptions{
		Args: args,
		Env:  env,
	})

	bp.GoToBuildDir()
	bp.RunCmakeBuild()
	bp.RunCmakeInstall([]string{"libogg" + libType.ToFilenameSuffix()})
}
