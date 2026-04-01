package ku

import (
	"github.com/mgenware/ku-builder/io2"
)

type StartEnvLoopOptions struct {
	// The main function to execute for each SDK/arch combination.
	LoopFn func(*BuildEnv)
	// Called before the loop starts, can be used for setup.
	BeforeAllFn func(*Shell)
	// Called after the loop ends, can be used for teardown.
	AfterAllFn func(*Shell)
	// When set, verifies file archs in the dist/lib directory after the loop.
	VerifyDistLibFileArch []string
	// When set, prevents automatic cleaning of the output directory before each loop iteration.
	DisableAutoClean bool
}

func StartEnvLoopWithOptions(cliOpt *CLIOptions, opt *StartEnvLoopOptions) {
	if opt == nil || opt.LoopFn == nil {
		panic("StartLoopWithOptions: LoopFn is required")
	}
	cliArgs := ParseCLIArgs(cliOpt)
	tunnel := CreateDefaultTunnel()
	shell := NewShell(tunnel, cliArgs)

	if opt.BeforeAllFn != nil {
		opt.BeforeAllFn(shell)
	}

	for _, sdk := range cliArgs.SDKs {
		var archs []ArchEnum
		if cliArgs.Arch != "" {
			archs = append(archs, cliArgs.Arch)
		} else {
			archs = SDKArchs[sdk]
		}

		for _, arch := range archs {
			osEnv := NewOSEnv(shell, sdk, arch)
			env := NewBuildEnv(shell, osEnv)

			if !opt.DisableAutoClean {
				io2.CleanDir(env.OutDir)
			}
			opt.LoopFn(env)

			if len(opt.VerifyDistLibFileArch) > 0 {
				env.VerifyDistLibFileArch(opt.VerifyDistLibFileArch)
			}
		}
	}

	if opt.AfterAllFn != nil {
		opt.AfterAllFn(shell)
	}
}

func StartEnvLoop(cliOpt *CLIOptions, fn func(*BuildEnv)) {
	StartEnvLoopWithOptions(cliOpt, &StartEnvLoopOptions{
		LoopFn: fn,
	})
}
