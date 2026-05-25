package zlib

import (
	"github.com/mgenware/ku-builder"
)

var repo = &ku.RepoInfo{
	Url:  "https://github.com/madler/zlib",
	Name: "zlib",
	Tag:  "v1.3.2",
}

func BuildZlib(be *ku.BuildEnv) {
	p := ku.NewCMakeProject(repo, be, ku.LibTypeStatic)
	p.Init(nil)
	p.Build()
	p.Install([]string{"libz.<s>"})
}
