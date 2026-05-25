package png

import (
	"github.com/mgenware/ku-builder"
)

var repo = &ku.RepoInfo{
	Url:  "https://github.com/pnggroup/libpng",
	Name: "libpng",
	Tag:  "v1.6.50",
}

func BuildPng(be *ku.BuildEnv) {
	p := ku.NewCMakeProject(repo, be, ku.LibTypeStatic)
	p.Init(&ku.ProjectInitOptions{
		Args: []string{
			"-DZLIB_INCLUDE_DIR=" + be.OutIncludeDir,
			"-DZLIB_LIBRARY=" + be.OutLibDir + "/libz.a",
		},
	})
	p.Build()
	p.Install([]string{"libpng.<s>"})
}
