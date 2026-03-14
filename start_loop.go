package ku

import (
	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type StartLoopOptions struct {
	ContextFn   func(*BuildContext)
	BeforeAllFn func(*CLIArgs, *j9.Tunnel)
	AfterAllFn  func(*CLIArgs, *j9.Tunnel)
}

func StartLoopWithOptions(libType LibType, cliOpt *CLIOptions, opt *StartLoopOptions) {
	if opt == nil || opt.ContextFn == nil {
		panic("StartLoopWithOptions: ContextFn is required")
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
			ctxOpt := NewBuildContextInitOpt(tunnel, sdk, arch, cliArgs, libType)
			ctx := NewBuildContext(ctxOpt)

			io2.CleanDir(ctx.OutDir)
			opt.ContextFn(ctx)
		}
	}

	if opt.AfterAllFn != nil {
		opt.AfterAllFn(cliArgs, tunnel)
	}
}

func StartLoop(libType LibType, cliOpt *CLIOptions, fn func(*BuildContext)) {
	StartLoopWithOptions(libType, cliOpt, &StartLoopOptions{
		ContextFn: fn,
	})
}
