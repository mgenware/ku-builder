package ku

import "strings"

type GetCompilerFlagsOptions struct {
	LD          bool
	DisableArch bool

	ExtraFlags []string
}

func (bp *Builder) GetCompilerFlagsList(opt *GetCompilerFlagsOptions) []string {
	if opt == nil {
		opt = &GetCompilerFlagsOptions{}
	}
	args := []string{}
	osEnv := bp.OS
	cliArgs := bp.Shell.Args

	if osEnv.IsDarwinPlatform() {
		archStr := string(osEnv.Arch)
		if !opt.DisableArch {
			args = append(args, "-arch", archStr)
		}

		args = append(args, "-isysroot", osEnv.GetSDKRootPath())

		// clang -target and min SDK version.
		args = append(args, "-target", osEnv.GetDarwinClangTargetTriple())

		switch osEnv.SDK {
		case SDKMacos:
			args = append(args, "-mmacosx-version-min="+MinMacosVersion)
		case SDKIosSimulator:
			args = append(args, "-mios-simulator-version-min="+MinIosVersion)
		case SDKIos:
			args = append(args, "-miphoneos-version-min="+MinIosVersion)
		}
	}

	if cliArgs.DebugBuild {
		args = append(args, "-g")
	}

	args = append(args, "-fPIC")

	if len(opt.ExtraFlags) > 0 {
		args = append(args, opt.ExtraFlags...)
	}

	return args
}

func (bp *Builder) GetCompilerFlagsString(opt *GetCompilerFlagsOptions) string {
	return strings.Join(bp.GetCompilerFlagsList(opt), " ")
}

type GetToolchainEnvOptions struct {
	// When true, override CFLAGS, CXXFLAGS, LDFLAGS.
	// Useful for make projects using `./configure`.
	// Note that might override existing compiler flags provided by source repo.
	// In that case, it's recommended to use `--extra-xxxflags` during `./configure`.
	// For CMake projects, these flags are passed via CMAKE_CXX_FLAGS, etc., so this option is not needed.
	// For Meson projects, these flags are passed via cross file, so this option is not needed.
	MakeOnlySetCompilerFlags bool

	MakeOnlyExtraCAndCXXFlags []string
	MakeOnlyExtraLDFlags      []string
}

func (bp *Builder) GetToolchainPathMap(buildSys BuildSystemEnum) [][]string {
	return bp.GetToolchainPathMapWithOptions(buildSys)
}

func (bp *Builder) GetToolchainPathMapWithOptions(buildSys BuildSystemEnum) [][]string {
	env := bp.OS

	if buildSys == BuildSystemCmake && env.IsAndroidPlatform() {
		// For CMake on Android, we have toolchain file that sets the compiler paths for us.
		return [][]string{}
	}

	lowerIfMeson := func(s string) string {
		if buildSys == BuildSystemMeson {
			return strings.ToLower(s)
		}
		return s
	}

	var cc string
	var cxx string
	var ld string

	switch buildSys {
	case BuildSystemMake:
		cc = "CC"
		cxx = "CXX"
		ld = "LD"
	case BuildSystemMeson:
		cc = "c"
		cxx = "cpp"
		ld = "ld"
	case BuildSystemCmake:
		cc = "CMAKE_C_COMPILER"
		cxx = "CMAKE_CXX_COMPILER"
		ld = "CMAKE_LINKER"
	}

	list := [][]string{
		{cc, env.GetCCPath()},
		{cxx, env.GetCXXPath()},
		{ld, env.GetLDPath()},
	}

	if env.IsDarwinPlatform() {
		var objc, objcpp string
		switch buildSys {
		case BuildSystemMake:
			objc = "OBJC"
			objcpp = "OBJCXX"
		case BuildSystemMeson:
			objc = "objc"
			objcpp = "objcpp"
		case BuildSystemCmake:
			objc = "CMAKE_OBJC_COMPILER"
			objcpp = "CMAKE_OBJCXX_COMPILER"
		}

		list = append(list, []string{objc, env.GetCCPath()})
		list = append(list, []string{objcpp, env.GetCXXPath()})
	}

	if env.IsAndroidPlatform() {
		// For Meson on Android, we also need to set other tools since there's no default
		// toolchain file that sets them for us like CMake.
		list = append(list, []string{lowerIfMeson("AR"), env.GetNDKToolchainBinPath("llvm-ar")})
		list = append(list, []string{lowerIfMeson("AS"), env.GetNDKToolchainBinPath("llvm-as")})
		list = append(list, []string{lowerIfMeson("NM"), env.GetNDKToolchainBinPath("llvm-nm")})
		list = append(list, []string{lowerIfMeson("RANLIB"), env.GetNDKToolchainBinPath("llvm-ranlib")})
		list = append(list, []string{lowerIfMeson("STRIP"), env.GetNDKToolchainBinPath("llvm-strip")})
	} else if env.IsDarwinPlatform() {
		list = append(list, []string{lowerIfMeson("AR"), env.RunXcodeFindCached("ar")})
		// In modern Xcode environments, there is no discrete as or separate assembler binary that you target directly. Apple relies entirely on Clang's integrated assembler.
		list = append(list, []string{lowerIfMeson("AS"), env.RunXcodeFindCached("clang")})
		list = append(list, []string{lowerIfMeson("NM"), env.RunXcodeFindCached("nm")})
		list = append(list, []string{lowerIfMeson("RANLIB"), env.RunXcodeFindCached("ranlib")})
		list = append(list, []string{lowerIfMeson("STRIP"), env.RunXcodeFindCached("strip")})
	}

	return list
}

// Returns environment variables for make project compiler configuration.
// This includes CC, CXX, LD, and optionally CFLAGS, CXXFLAGS, LDFLAGS (when OverrideCompilerFlags is true).
func (bp *Builder) GetMakeToolchainEnv(opt *GetToolchainEnvOptions) []string {
	if opt == nil {
		opt = &GetToolchainEnvOptions{}
	}

	args := []string{}

	toolchainPathMap := bp.GetToolchainPathMap(BuildSystemMake)
	for _, pair := range toolchainPathMap {
		args = append(args, pair[0]+"="+pair[1])
	}

	if opt.MakeOnlySetCompilerFlags {
		cflags := bp.GetCompilerFlagsString(&GetCompilerFlagsOptions{
			ExtraFlags: opt.MakeOnlyExtraCAndCXXFlags,
		})
		ldflags := bp.GetCompilerFlagsString(&GetCompilerFlagsOptions{
			ExtraFlags: opt.MakeOnlyExtraLDFlags,
			LD:         true,
		})

		args = append(args, "CFLAGS="+cflags)
		args = append(args, "CXXFLAGS="+cflags)
		args = append(args, "LDFLAGS="+ldflags)
	}

	return args
}
