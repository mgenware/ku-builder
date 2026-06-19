package ku

import (
	"fmt"
	"runtime"

	"github.com/mgenware/j9/v3"
)

func (bp *Builder) RunMakeCleanRaw() error {
	env := bp.GetKuBuiltinEnv(false)
	return bp.BuildEnv.Shell.SpawnRaw(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"clean"},
		Env:  env,
	})
}

func (bp *Builder) RunMakeClean() {
	err := bp.RunMakeCleanRaw()
	if err != nil {
		panic(err)
	}
}

func (bp *Builder) RunMakeWithArgs(opt *j9.SpawnOpt) {
	if opt == nil {
		opt = &j9.SpawnOpt{}
	}
	numCores := runtime.NumCPU()

	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(false), opt.Env...)

	bp.BuildEnv.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{fmt.Sprintf("-j%v", numCores)},
		Env:  env,
	})
}

func (bp *Builder) RunMake() {
	bp.RunMakeWithArgs(nil)
}

func (bp *Builder) RunMakeInstall(outFile string, vfOpt *VerifyFileOptions) {
	env := bp.GetKuBuiltinEnv(false)
	bp.BuildEnv.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"install"},
		Env:  env,
	})
	bp.BuildEnv.VerifyFile(outFile, vfOpt)
}
