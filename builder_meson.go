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

type GetMesonSetupArgsOptions struct {
	// If true, run `meson configure` instead of `meson setup`.
	Configure  bool
	CleanBuild bool
}

func (bp *Builder) GetMesonSetupArgs() []string {
	return bp.GetMesonSetupArgsWithOptions(nil)
}

func (bp *Builder) GetMesonSetupArgsWithOptions(opt *GetMesonSetupArgsOptions) []string {
	if opt == nil {
		opt = &GetMesonSetupArgsOptions{}
	}

	rootAction := "setup"
	if opt.Configure {
		rootAction = "configure"
	}
	args := []string{rootAction}
	cliArgs := bp.CLIArgs
	buildEnv := bp.BuildEnv
	libType := bp.LibType

	if cliArgs.CleanBuild || opt.CleanBuild {
		args = append(args, "--wipe")
	}
	var buildType string
	if cliArgs.DebugBuild {
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

	args = append(args, "--prefix="+buildEnv.OutDir)
	args = append(args, "--cmake-prefix-path="+buildEnv.OutDir)

	crossFilePath, err := bp.getOrCreateCrossFilePath()
	if err != nil {
		bp.Shell.Quit(fmt.Sprintf("Failed to create Meson cross file: %v", err))
		return nil
	}
	if !opt.Configure {
		args = append(args, "--cross-file="+crossFilePath)
	}

	// Append the build dir as the last argument.
	args = append(args, bp.mustGetBuildDir(opt.CleanBuild))
	return args
}

type RunMesonSetupOptions struct {
	Args []string
	Env  []string
}

func (bp *Builder) RunMesonSetup(opt *RunMesonSetupOptions) {
	bp.NotNullOrQuit(opt, "opt")

	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(true), opt.Env...)

	bp.Shell.Spawn(&j9.SpawnOpt{
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

func (bp *Builder) RunMesonBuildOrInstall(opt *RunMesonBuildOrInstallOptions, outFile []string) {
	bp.NotNullOrQuit(opt, "opt")
	bp.NotNullOrQuit(opt.Action, "opt.Action")

	cliArgs := bp.CLIArgs

	args := []string{
		string(opt.Action),
	}

	// Strip during production install.
	if opt.Action == MesonActionInstall && !cliArgs.DebugBuild {
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
			bp.Shell.Quit("opt.Target is not supported for install")
		}
		args = append(args, opt.Target)
	}

	// Note: `opt.Env` should be set after `GetKuBuiltinEnv`.
	env := append(bp.GetKuBuiltinEnv(false), opt.Env...)
	env = append(env,
		"KU_MESON_ACTION="+string(opt.Action),
	)
	bp.Shell.Spawn(&j9.SpawnOpt{
		Name: "meson",
		Args: args,
		Env:  env,
	})

	bp.BuildEnv.VerifyLibFileArch(outFile)
}

func (bp *Builder) RunMesonCompile() {
	bp.RunMesonCompileTarget("")
}

func (bp *Builder) RunMesonCompileTarget(target string) {
	opt := &RunMesonBuildOrInstallOptions{
		Action: MesonActionCompile,
		Target: target,
	}
	bp.RunMesonBuildOrInstall(opt, nil)
}

func (bp *Builder) RunMesonInstall(outFile []string) {
	opt := &RunMesonBuildOrInstallOptions{
		Action: MesonActionInstall,
	}
	bp.RunMesonBuildOrInstall(opt, outFile)
}

func (bp *Builder) getOrCreateCrossFilePath() (string, error) {
	key := bp.OS.GetSDKArchString()
	if path, ok := mesonCrossFileCache[key]; ok {
		return path, nil
	}
	path, err := bp.writeCrossFile()
	if err != nil {
		return "", err
	}
	mesonCrossFileCache[key] = path
	return path, nil
}

func (bp *Builder) writeCrossFile() (string, error) {
	paths := []string{kMesonCrossFileDir, bp.OS.GetSDKArchString() + ".txt"}
	content := bp.createCrossFile()
	path, err := util.WriteKuCacheFile(content, paths)
	if err != nil {
		return "", err
	}
	return path, nil
}

// Important! Don't add project-specific options because cross files are cached based on OS and architecture.
func (bp *Builder) createCrossFile() string {
	var sb strings.Builder
	osEnv := bp.OS
	be := bp.BuildEnv

	sb.WriteString("[binaries]\n")
	compilerPathMap := bp.GetToolchainPathMapWithOptions(&GetToolchainPathMapOptions{Meson: true})
	for _, pair := range compilerPathMap {
		sb.WriteString(strings.ToLower(pair[0]) + " = '" + pair[1] + "'\n")
	}

	sb.WriteString("[built-in options]\n")
	cflags := bp.GetCompilerFlagsList(nil)
	ldflags := bp.GetCompilerFlagsList(&GetCompilerFlagsOptions{LD: true})
	sb.WriteString("c_args = " + joinStringsWithSingleQuotes(cflags) + "\n")
	sb.WriteString("cpp_args = " + joinStringsWithSingleQuotes(cflags) + "\n")
	sb.WriteString("c_link_args = " + joinStringsWithSingleQuotes(ldflags) + "\n")
	sb.WriteString("cpp_link_args = " + joinStringsWithSingleQuotes(ldflags) + "\n")
	sb.WriteString("b_staticpic = true\n")

	sb.WriteString("[properties]\n")
	sb.WriteString("sys_root = '" + osEnv.GetSDKRootPath() + "'\n")

	// TODO: disable crossfile cache.
	pkgConfigLibDir := filepath.Join(be.OutDir, "lib", "pkgconfig")
	sb.WriteString("pkg_config_libdir = '" + pkgConfigLibDir + "'\n")

	// Required when cross-compiling for a different architecture (e.g., x86_64 to ARM) and your build system cannot directly execute the generated target binaries.
	sb.WriteString("needs_exe_wrapper = true\n")

	sb.WriteString("[host_machine]\n")
	var system string
	isAndroid := osEnv.IsAndroidPlatform()
	if isAndroid {
		system = "android"
	} else {
		system = "darwin"
	}

	var subSystem string
	if osEnv.SDK == SDKIosSimulator {
		subSystem = "ios-simulator"
	}

	var cpu string
	switch osEnv.Arch {
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
