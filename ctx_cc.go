package ku

import "strings"

type GetCompilerFlagsOptions struct {
	LD          bool
	DisableArch bool
	EnablePIC   bool
}

func (ctx *BuildContext) getCompilerFlagsList(opt *GetCompilerFlagsOptions) []string {
	if opt == nil {
		opt = &GetCompilerFlagsOptions{}
	}
	args := []string{}

	if ctx.Env.IsDarwinPlatform() {
		archStr := string(ctx.Arch)
		if !opt.DisableArch {
			args = append(args, "-arch", archStr)
		}

		args = append(args, "-isysroot", ctx.Env.GetSDKRootPath())

		// clang -target and min SDK version.
		args = append(args, "-target", ctx.Env.GetDarwinClangTargetTriple())

		switch ctx.SDK {
		case SDKMacos:
			args = append(args, "-mmacosx-version-min="+MinMacosVersion)
		case SDKIosSimulator:
			args = append(args, "-mios-simulator-version-min="+MinIosVersion)
		case SDKIos:
			args = append(args, "-miphoneos-version-min="+MinIosVersion)
		}
	}

	if ctx.DebugBuild {
		args = append(args, "-g")
	}

	if opt.EnablePIC {
		args = append(args, "-fPIC")
	}

	return args
}

func (ctx *BuildContext) GetCompilerFlags(opt *GetCompilerFlagsOptions) string {
	return strings.Join(ctx.getCompilerFlagsList(opt), " ")
}

type GetCompilerConfigureEnvOptions struct {
	// When true, override CFLAGS, CXXFLAGS, LDFLAGS.
	// Useful for make projects using `./configure`.
	// Note that might override existing compiler flags provided by source repo.
	// In that case, it's recommended to use `--extra-xxxflags` during `./configure`.
	OverrideCompilerFlags bool
}

// GetCompilerConfigureEnv returns environment variables for compiler configuration.
// This includes CC, CXX, LD, and optionally CFLAGS, CXXFLAGS, LDFLAGS (when OverrideCompilerFlags is true).
// On Android, it also includes AR, AS, RANLIB, STRIP, NM.
func (ctx *BuildContext) GetCompilerConfigureEnv(opt *GetCompilerConfigureEnvOptions) []string {
	if opt == nil {
		opt = &GetCompilerConfigureEnvOptions{}
	}

	args := []string{
		"CC=" + ctx.Env.GetCCPath(),
		"CXX=" + ctx.Env.GetCXXPath(),
		"LD=" + ctx.Env.GetLDPath(),
	}
	if ctx.Env.IsAndroidPlatform() {
		args = append(args, "AR="+ctx.Env.GetNDKToolchainBinPath("llvm-ar"))
		args = append(args, "AS="+ctx.Env.GetNDKToolchainBinPath("llvm-as"))
		args = append(args, "RANLIB="+ctx.Env.GetNDKToolchainBinPath("llvm-ranlib"))
		args = append(args, "STRIP="+ctx.Env.GetNDKToolchainBinPath("llvm-strip"))
		args = append(args, "NM="+ctx.Env.GetNDKToolchainBinPath("llvm-nm"))
	}

	if opt.OverrideCompilerFlags {
		cflags := ctx.GetCompilerFlags(nil)
		ldflags := ctx.GetCompilerFlags(&GetCompilerFlagsOptions{LD: true})

		args = append(args, "CFLAGS="+cflags)
		args = append(args, "CXXFLAGS="+cflags)
		args = append(args, "LDFLAGS="+ldflags)
	}

	return args
}
