package ku

import "strings"

type GetCompilerFlagsOptions struct {
	LD          bool
	DisableArch bool
	EnablePIC   bool
}

func (bp *BuildProject) GetCompilerFlagList(opt *GetCompilerFlagsOptions) []string {
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

	if opt.EnablePIC {
		args = append(args, "-fPIC")
	}

	return args
}

func (bp *BuildProject) GetCompilerFlags(opt *GetCompilerFlagsOptions) string {
	return strings.Join(bp.GetCompilerFlagList(opt), " ")
}

type GetCompilerConfigureEnvOptions struct {
	// When true, override CFLAGS, CXXFLAGS, LDFLAGS.
	// Useful for make projects using `./configure`.
	// Note that might override existing compiler flags provided by source repo.
	// In that case, it's recommended to use `--extra-xxxflags` during `./configure`.
	OverrideCompilerFlags bool
}

type GetCompilerPathMapOptions struct {
	Meson bool
}

func (bp *BuildProject) GetCompilerPathMap() [][]string {
	return bp.GetCompilerPathMapWithOptions(nil)
}

func (bp *BuildProject) GetCompilerPathMapWithOptions(opt *GetCompilerPathMapOptions) [][]string {
	if opt == nil {
		opt = &GetCompilerPathMapOptions{}
	}
	env := bp.OS
	isDarwin := env.IsDarwinPlatform()

	cc := "CC"
	if opt.Meson {
		cc = "C"
	}
	cxx := "CXX"
	if opt.Meson {
		cxx = "CPP"
	}

	list := [][]string{
		{cc, env.GetCCPath()},
		{cxx, env.GetCXXPath()},
		{"LD", env.GetLDPath()},
	}

	if isDarwin {
		objCXX := "OBJCXX"
		if opt.Meson {
			objCXX = "OBJCPP"
		}
		list = append(list, []string{"OBJC", env.GetCCPath()})
		list = append(list, []string{objCXX, env.GetCXXPath()})
	}

	if env.IsAndroidPlatform() {
		list = append(list, []string{"AR", env.GetNDKToolchainBinPath("llvm-ar")})
		list = append(list, []string{"AS", env.GetNDKToolchainBinPath("llvm-as")})
		list = append(list, []string{"RANLIB", env.GetNDKToolchainBinPath("llvm-ranlib")})
		list = append(list, []string{"STRIP", env.GetNDKToolchainBinPath("llvm-strip")})
		list = append(list, []string{"NM", env.GetNDKToolchainBinPath("llvm-nm")})
	}
	return list
}

// GetCompilerConfigureEnv returns environment variables for compiler configuration.
// This includes CC, CXX, LD, and optionally CFLAGS, CXXFLAGS, LDFLAGS (when OverrideCompilerFlags is true).
// On Android, it also includes AR, AS, RANLIB, STRIP, NM.
func (bp *BuildProject) GetCompilerConfigureEnv(opt *GetCompilerConfigureEnvOptions) []string {
	if opt == nil {
		opt = &GetCompilerConfigureEnvOptions{}
	}

	args := []string{}

	compilerPathMap := bp.GetCompilerPathMap()
	for _, pair := range compilerPathMap {
		args = append(args, pair[0]+"="+pair[1])
	}

	if opt.OverrideCompilerFlags {
		cflags := bp.GetCompilerFlags(nil)
		ldflags := bp.GetCompilerFlags(&GetCompilerFlagsOptions{LD: true})

		args = append(args, "CFLAGS="+cflags)
		args = append(args, "CXXFLAGS="+cflags)
		args = append(args, "LDFLAGS="+ldflags)
	}

	return args
}
