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

	var cc string
	var cxx string
	var ld string

	switch buildSys {
	case BuildSystemMake:
		cc = "CC"
		cxx = "CXX"
		ld = "LD"
	case BuildSystemMeson:
		cc = "C"
		cxx = "CPP"
		ld = "LD"
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
		objC := "OBJC"
		if buildSys == BuildSystemCmake {
			objC = "CMAKE_OBJC_COMPILER"
		}

		objCXX := "OBJCXX"
		if buildSys == BuildSystemMeson {
			objCXX = "OBJCPP"
		} else if buildSys == BuildSystemCmake {
			objCXX = "CMAKE_OBJCXX_COMPILER"
		}
		list = append(list, []string{objC, env.GetCCPath()})
		list = append(list, []string{objCXX, env.GetCXXPath()})
	}

	if env.IsAndroidPlatform() {
		// For Meson on Android, we also need to set other tools since there's no default
		// toolchain file that sets them for us like CMake.
		list = append(list, []string{"AR", env.GetNDKToolchainBinPath("llvm-ar")})
		list = append(list, []string{"AS", env.GetNDKToolchainBinPath("llvm-as")})
		list = append(list, []string{"NM", env.GetNDKToolchainBinPath("llvm-nm")})
		list = append(list, []string{"RANLIB", env.GetNDKToolchainBinPath("llvm-ranlib")})
		list = append(list, []string{"STRIP", env.GetNDKToolchainBinPath("llvm-strip")})
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
