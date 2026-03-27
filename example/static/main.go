package main

import (
	"github.com/mgenware/ku-builder"
	"github.com/mgenware/ku-builder/example"
)

func main() {
	cliOpt := &ku.CLIOptions{
		DefaultTarget: example.LibName,
	}
	libType := ku.LibTypeStatic
	ku.StartEnvLoop(cliOpt, func(be *ku.BuildEnv) {
		be.LogSummary()
		example.BuildOgg(be, libType)
	})
}
