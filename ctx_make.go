package ku

import (
	"fmt"
	"runtime"

	"github.com/mgenware/j9/v3"
)

func (ctx *BuildContext) RunMakeCleanRaw() error {
	env := ctx.GetCoreKuEnv()
	return ctx.Shell.SpawnRaw(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"clean"},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMakeClean() {
	err := ctx.RunMakeCleanRaw()
	if err != nil {
		panic(err)
	}
}

func (ctx *BuildContext) RunMakeWithArgs(opt *j9.SpawnOpt) {
	if opt == nil {
		opt = &j9.SpawnOpt{}
	}
	numCores := runtime.NumCPU()

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)

	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{fmt.Sprintf("-j%v", numCores)},
		Env:  env,
	})
}

func (ctx *BuildContext) RunMake() {
	ctx.RunMakeWithArgs(nil)
}

func (ctx *BuildContext) RunMakeInstall(outFile []string) {
	env := ctx.GetCoreKuEnv()
	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"install"},
		Env:  env,
	})
	ctx.VerifyOutLibFileArch(outFile)
}
