package ku

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/util"
)

const kMesonCrossFileDir = "meson_cross_files"

// K: `Env.GetSDKArchString()`, V: cached cross file path.
var mesonCrossFileCache = make(map[string]string)

func (ctx *BuildContext) GetMesonSetupArgs(libType LibType, buildDir string) []string {
	args := []string{
		"setup",
	}
	if ctx.CleanBuild {
		args = append(args, "--wipe")
	}
	var buildType string
	if ctx.DebugBuild {
		buildType = "debug"
		args = append(args, "--debug")
	} else {
		buildType = "release"
	}
	args = append(args, "--buildtype="+buildType)

	var libTypeArg string
	if libType == LibTypeStatic {
		libTypeArg = "static"
	} else {
		libTypeArg = "shared"
	}
	args = append(args, "--default-library="+libTypeArg)

	args = append(args, "--prefix="+ctx.OutDir)
	args = append(args, "--cmake-prefix-path="+ctx.OutDir)

	pkgConfigPath := filepath.Join(ctx.OutDir, "lib", "pkgconfig")
	args = append(args, "--pkg-config-path="+pkgConfigPath)

	crossFilePath, err := ctx.getOrCreateCrossFilePath()
	if err != nil {
		ctx.Shell.Quit(fmt.Sprintf("Failed to create Meson cross file: %v", err))
		return nil
	}
	args = append(args, "--cross-file="+crossFilePath)

	// Append the build dir as the last argument.
	args = append(args, buildDir)
	return args
}

type RunMesonSetupOptions struct {
	Args []string
	Env  []string
}

func (ctx *BuildContext) RunMesonSetup(opt *RunMesonSetupOptions) {
	ctx.NotNullOrQuit(opt, "opt")

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)

	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "meson",
		Args: opt.Args,
		Env:  env,
	})
}

type MesonActionType string

const (
	MesonActionCompile MesonActionType = "compile"
	MesonActionInstall MesonActionType = "install"
)

type RunMesonBuildOrInstallOptions struct {
	// Required.
	Action MesonActionType

	Target    string
	ExtraArgs []string
	Env       []string
}

func (ctx *BuildContext) RunMesonBuildOrInstall(opt *RunMesonBuildOrInstallOptions, outFile []string) {
	ctx.NotNullOrQuit(opt, "opt")
	ctx.NotNullOrQuit(opt.Action, "opt.Action")

	args := []string{
		string(opt.Action),
	}

	// Strip during production install.
	if opt.Action == MesonActionInstall && !ctx.DebugBuild {
		args = append(args, "--strip")
	}

	if opt.Action == MesonActionCompile {
		numCores := runtime.NumCPU()
		args = append(args, "-j", fmt.Sprintf("%v", numCores))
	}

	// Extra args.
	if len(opt.ExtraArgs) > 0 {
		args = append(args, opt.ExtraArgs...)
	}

	// Target is the last argument.
	if opt.Target != "" {
		if opt.Action == MesonActionInstall {
			ctx.Shell.Quit("opt.Target is not supported for install")
		}
		args = append(args, opt.Target)
	}

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)
	env = append(env,
		"KU_MESON_ACTION="+string(opt.Action),
	)
	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "meson",
		Args: args,
		Env:  env,
	})

	ctx.VerifyOutLibFileArch(outFile)
}

func (ctx *BuildContext) RunMesonCompile() {
	ctx.RunMesonCompileTarget("")
}

func (ctx *BuildContext) RunMesonCompileTarget(target string) {
	opt := &RunMesonBuildOrInstallOptions{
		Action: MesonActionCompile,
		Target: target,
	}
	ctx.RunMesonBuildOrInstall(opt, nil)
}

func (ctx *BuildContext) RunMesonInstall(outFile []string) {
	opt := &RunMesonBuildOrInstallOptions{
		Action: MesonActionInstall,
	}
	ctx.RunMesonBuildOrInstall(opt, outFile)
}

func (ctx *BuildContext) getOrCreateCrossFilePath() (string, error) {
	key := ctx.Env.GetSDKArchString()
	if path, ok := mesonCrossFileCache[key]; ok {
		return path, nil
	}
	path, err := ctx.writeCrossFile()
	if err != nil {
		return "", err
	}
	mesonCrossFileCache[key] = path
	return path, nil
}

func (ctx *BuildContext) writeCrossFile() (string, error) {
	paths := []string{kMesonCrossFileDir, ctx.Env.GetSDKArchString() + ".txt"}
	content := ctx.createCrossFile()
	path, err := util.WriteKuCacheFile(content, paths)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (ctx *BuildContext) createCrossFile() string {
	var sb strings.Builder

	sb.WriteString("[binaries]\n")
	compilerPathMap := ctx.GetCompilerPathMapWithOptions(&GetCompilerPathMapOptions{Meson: true})
	for _, pair := range compilerPathMap {
		sb.WriteString(strings.ToLower(pair[0]) + " = '" + pair[1] + "'\n")
	}

	sb.WriteString("[built-in options]\n")
	cflags := ctx.GetCompilerFlagList(nil)
	ldflags := ctx.GetCompilerFlagList(&GetCompilerFlagsOptions{LD: true})
	sb.WriteString("c_args = " + joinStringsWithSingleQuotes(cflags) + "\n")
	sb.WriteString("cpp_args = " + joinStringsWithSingleQuotes(cflags) + "\n")
	sb.WriteString("c_link_args = " + joinStringsWithSingleQuotes(ldflags) + "\n")
	sb.WriteString("cpp_link_args = " + joinStringsWithSingleQuotes(ldflags) + "\n")

	sb.WriteString("[properties]\n")
	sb.WriteString("needs_exe_wrapper = true\n")
	sb.WriteString("root = '" + ctx.Env.GetSDKRootPath() + "'\n")

	sb.WriteString("[host_machine]\n")
	var system string
	isAndroid := ctx.Env.IsAndroidPlatform()
	if isAndroid {
		system = "android"
	} else {
		system = "darwin"
	}

	var subSystem string
	if ctx.SDK == SDKIosSimulator {
		subSystem = "ios-simulator"
	}

	var cpu string
	switch ctx.Arch {
	case ArchArm64:
		cpu = "aarch64"
	case ArchX86_64:
		cpu = "x86_64"
	}

	sb.WriteString(fmt.Sprintf("system = '%s'\n", system))
	if subSystem != "" {
		sb.WriteString(fmt.Sprintf("subsystem = '%s'\n", subSystem))
	}
	sb.WriteString(fmt.Sprintf("cpu_family = '%s'\n", cpu))
	sb.WriteString(fmt.Sprintf("cpu = '%s'\n", cpu))
	sb.WriteString("endian = 'little'\n")

	return sb.String()
}

func joinStringsWithSingleQuotes(list []string) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, s := range list {
		sb.WriteString(fmt.Sprintf("'%s'", s))
		if i != len(list)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}
