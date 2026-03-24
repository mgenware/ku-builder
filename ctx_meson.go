package ku

import (
	"fmt"
	"strings"
)

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
