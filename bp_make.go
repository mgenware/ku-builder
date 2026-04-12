package ku

import (
	"fmt"
	"runtime"

	"github.com/mgenware/j9/v3"
)

func (bp *BuildProject) RunMakeCleanRaw() error {
	env := bp.GetKuBuiltinEnv()
	return bp.BuildEnv.Shell.SpawnRaw(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"clean"},
		Env:  env,
	})
}

func (bp *BuildProject) RunMakeClean() {
	err := bp.RunMakeCleanRaw()
	if err != nil {
		panic(err)
	}
}

func (bp *BuildProject) RunMakeWithArgs(opt *j9.SpawnOpt) {
	if opt == nil {
		opt = &j9.SpawnOpt{}
	}
	numCores := runtime.NumCPU()

	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(), opt.Env...)

	bp.BuildEnv.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{fmt.Sprintf("-j%v", numCores)},
		Env:  env,
	})
}

func (bp *BuildProject) RunMake() {
	bp.RunMakeWithArgs(nil)
}

func (bp *BuildProject) RunMakeInstall(outFile []string) {
	env := bp.GetKuBuiltinEnv()
	bp.BuildEnv.Shell.Spawn(&j9.SpawnOpt{
		Name: "make",
		Args: []string{"install"},
		Env:  env,
	})
	bp.BuildEnv.VerifyLibFileArch(outFile)
}
