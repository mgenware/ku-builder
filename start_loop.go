package ku

import "github.com/mgenware/j9/v3"

type StartLoopOptions struct {
	ContextFn   func(*BuildContext)
	BeforeAllFn func(*CLIArgs, *j9.Tunnel)
	AfterAllFn  func(*CLIArgs, *j9.Tunnel)
}

func StartLoopWithOptions(cliOpt *CLIOptions, opt *StartLoopOptions) {
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
			ctxOpt := NewBuildContextInitOpt(tunnel, sdk, arch, cliArgs)
			ctx := NewBuildContext(ctxOpt)

			opt.ContextFn(ctx)
		}
	}

	if opt.AfterAllFn != nil {
		opt.AfterAllFn(cliArgs, tunnel)
	}
}

func StartLoop(cliOpt *CLIOptions, fn func(*BuildContext)) {
	StartLoopWithOptions(cliOpt, &StartLoopOptions{
		ContextFn: fn,
	})
}
