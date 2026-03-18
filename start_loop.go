package ku

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type StartLoopOptions struct {
	// The main function to execute for each SDK/arch combination.
	LoopFn func(*BuildContext)
	// Called before the loop starts, can be used for setup.
	BeforeAllFn func(*CLIArgs, *j9.Tunnel)
	// Called after the loop ends, can be used for teardown.
	AfterAllFn func(*CLIArgs, *j9.Tunnel)
	// When set, verifies file archs in the dist directory after the loop.
	VerifyDistFileArch []string
}

func StartLoopWithOptions(cliOpt *CLIOptions, opt *StartLoopOptions) {
	if opt == nil || opt.LoopFn == nil {
		panic("StartLoopWithOptions: LoopFn is required")
	}
	cliArgs := ParseCLIArgs(cliOpt)
	tunnel := CreateDefaultTunnel()

	if opt.BeforeAllFn != nil {
		opt.BeforeAllFn(cliArgs, tunnel)
	}

	for _, sdk := range cliArgs.SDKs {
		var archs []ArchEnum
		if cliArgs.Arch != "" {
			archs = append(archs, cliArgs.Arch)
		} else {
			archs = SDKArchs[sdk]
		}

		for _, arch := range archs {
			ctxOpt := NewBuildContextInitOpt(tunnel, sdk, arch, cliArgs)
			ctx := NewBuildContext(ctxOpt)

			io2.CleanDir(ctx.OutDir)
			opt.LoopFn(ctx)

			if len(opt.VerifyDistFileArch) > 0 {
				ctx.VerifyDistFileArch(opt.VerifyDistFileArch)
			}
		}
	}

	if opt.AfterAllFn != nil {
		opt.AfterAllFn(cliArgs, tunnel)
	}
}

func StartLoop(cliOpt *CLIOptions, fn func(*BuildContext)) {
	StartLoopWithOptions(cliOpt, &StartLoopOptions{
		LoopFn: fn,
	})
}
