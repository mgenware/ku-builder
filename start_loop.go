package ku

func StartLoop(cliOpt *CLIOptions, fn func(*BuildContext)) {
	cliArgs := ParseCLIArgs(cliOpt)
	tunnel := CreateDefaultTunnel()

	for _, sdk := range cliArgs.SDKs {
		var archs []ArchEnum
		if cliArgs.Arch != "" {
			archs = append(archs, cliArgs.Arch)
		} else {
			archs = SDKArchs[sdk]
		}

		for _, arch := range archs {
			opt := NewBuildContextInitOpt(tunnel, sdk, arch, cliArgs)
			ctx := NewBuildContext(opt)

			fn(ctx)
		}
	}
}
