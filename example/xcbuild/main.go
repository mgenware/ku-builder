package main

import (
	"github.com/mgenware/ku-builder/xcbuild"
)

const kTarget = "libogg"

func main() {
	opt := &xcbuild.XCBuildOptions{
		DefaultTarget: kTarget,
		GetModuleMapTargets: func(ctx *xcbuild.XCContext) []string {
			return []string{kTarget}
		},
	}

	xcbuild.Build(opt)
}
