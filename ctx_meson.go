package ku

import (
	"fmt"
	"strings"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/util"
)

const kMesonCrossFileDir = "meson_cross_files"

var mesonCrossFileCache = make(map[SDKEnum]string)

type RunMesonSetupOptions struct {
	Args []string
	Env  []string
}

func (ctx *BuildContext) RunMesonSetup(opt *RunMesonSetupOptions) {
	args := []string{
		"setup",
	}
	// Add `opt.Args` after `setup`.
	args = append(args, opt.Args...)

	if ctx.CleanBuild {
		args = append(args, "--wipe")
	}
	var buildType string
	if ctx.DebugBuild {
		buildType = "debug"
	} else {
		buildType = "release"
	}
	args = append(args, "--buildtype="+buildType)

	crossFilePath, err := ctx.getOrCreateCrossFilePath()
	if err != nil {
		ctx.Shell.Quit(fmt.Sprintf("Failed to create Meson cross file: %v", err))
		return
	}
	args = append(args, "--cross-file="+crossFilePath)

	// Note: `opt.Env` should be set after `GetCoreKuEnv`.
	env := append(ctx.GetCoreKuEnv(), opt.Env...)

	ctx.Shell.Spawn(&j9.SpawnOpt{
		Name: "meson",
		Args: args,
		Env:  env,
	})
}

func (ctx *BuildContext) getOrCreateCrossFilePath() (string, error) {
	if path, ok := mesonCrossFileCache[ctx.SDK]; ok {
		return path, nil
	}
	path, err := ctx.writeCrossFile()
	if err != nil {
		return "", err
	}
	mesonCrossFileCache[ctx.SDK] = path
	return path, nil
}

func (ctx *BuildContext) writeCrossFile() (string, error) {
	paths := []string{kMesonCrossFileDir, string(ctx.SDK) + ".txt"}
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
	compilerPathMap := ctx.GetCompilerPathMap()
	for _, pair := range compilerPathMap {
		sb.WriteString(pair[0] + " = '" + pair[1] + "'\n")
	}

	sb.WriteString("[built-in options]\n")
	cflags := ctx.GetCompilerFlagList(nil)
	ldflags := ctx.GetCompilerFlagList(&GetCompilerFlagsOptions{LD: true})
	sb.WriteString("c_args = " + jsonfyStringList(cflags) + "\n")
	sb.WriteString("cpp_args = " + jsonfyStringList(cflags) + "\n")
	sb.WriteString("c_link_args = " + jsonfyStringList(ldflags) + "\n")
	sb.WriteString("cpp_link_args = " + jsonfyStringList(ldflags) + "\n")

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

func jsonfyStringList(list []string) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, s := range list {
		sb.WriteString(fmt.Sprintf("%q", s))
		if i != len(list)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}
