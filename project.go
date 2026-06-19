package ku

import (
	"fmt"
	"path/filepath"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder/io2"
)

type ProjectInitOptions struct {
	Args []string
	Env  []string

	// Cmake options.
	GetCmakeSetupArgsOptions *GetCmakeGenArgsOptions
	RunCmakeSetupOptions     *RunCmakeGenOptions

	// Meson options.
	GetMesonSetupArgsOptions *GetMesonSetupArgsOptions
	RunMesonSetupOptions     *RunMesonSetupOptions

	// Make options.
	MakeExtraCAndCXXFlags []string
	MakeExtraLDFlags      []string
}

type Project interface {
	// This calls `CloneAndGotoRepoSource` first, then runs the project setup command.
	Init(opt *ProjectInitOptions)

	// Starts the build process. This should be called after `Init`.
	Build()

	// Installs the built library to the output directory. This should be called after `Build`.
	Install(outFile string, vfOpt *VerifyFileOptions)

	// Returns the core builder for this project. This is used for advanced users who want to run custom commands.
	CoreBuilder() *Builder
}

type CMakeProject struct {
	builder *Builder
}

func NewCMakeProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) Project {
	builder := NewBuilder(repo, buildEnv, libType)
	return &CMakeProject{
		builder: builder,
	}
}

func (p *CMakeProject) CoreBuilder() *Builder {
	return p.builder
}

func (p *CMakeProject) Init(opt *ProjectInitOptions) {
	if opt == nil {
		opt = &ProjectInitOptions{}
	}

	b := p.builder
	b.CloneAndGotoRepoSource()

	args := b.GetCmakeGenArgsWithOptions(opt.GetCmakeSetupArgsOptions)
	if len(opt.Args) > 0 {
		args = append(args, opt.Args...)
	}

	env := []string{}
	if len(opt.Env) > 0 {
		env = append(env, opt.Env...)
	}

	var genOpt *RunCmakeGenOptions
	// Merge options.
	if opt.RunCmakeSetupOptions != nil {
		genOpt = opt.RunCmakeSetupOptions
		genOpt.Args = append(args, genOpt.Args...)
		genOpt.Env = append(env, genOpt.Env...)
	} else {
		genOpt = &RunCmakeGenOptions{
			Args: args,
			Env:  env,
		}
	}

	b.RunCmakeGen(genOpt)
}

func (p *CMakeProject) Build() {
	b := p.builder
	b.GoToBuildDir()
	b.RunCmakeBuild()
}

func (p *CMakeProject) Install(outFile string, vfOpt *VerifyFileOptions) {
	b := p.builder
	b.RunCmakeInstall(outFile, vfOpt)
}

type MakeProject struct {
	builder *Builder
}

func (p *MakeProject) CoreBuilder() *Builder {
	return p.builder
}

func NewMakeProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) Project {
	builder := NewBuilder(repo, buildEnv, libType)
	return &MakeProject{
		builder: builder,
	}
}

func (p *MakeProject) Init(opt *ProjectInitOptions) {
	if opt == nil {
		opt = &ProjectInitOptions{}
	}

	b := p.builder
	srcDir := b.CloneAndGotoRepoSource()

	env := b.GetMakeToolchainEnv(&GetToolchainEnvOptions{
		MakeOnlySetCompilerFlags:  true,
		MakeOnlyExtraCAndCXXFlags: opt.MakeExtraCAndCXXFlags,
		MakeOnlyExtraLDFlags:      opt.MakeExtraLDFlags,
	})

	// Note: `opt.Env` should come at last to allow overriding builtin env if needed.
	env = append(env, b.GetKuBuiltinEnv(true)...)
	env = append(env, opt.Env...)

	// Run ./configure at build dir, not source dir.
	b.GoToBuildDir()
	configureFilePath := filepath.Join(srcDir, "configure")
	if !io2.FileExists(configureFilePath) {
		b.Shell.Quit(fmt.Sprintf("configure script not found at %s", configureFilePath))
	}
	b.Shell.Spawn(&j9.SpawnOpt{
		Name: configureFilePath,
		Args: opt.Args,
		Env:  env,
	})
}

func (p *MakeProject) Build() {
	b := p.builder
	b.GoToBuildDir()
	b.RunMake()
}

func (p *MakeProject) Install(outFile string, vfOpt *VerifyFileOptions) {
	b := p.builder
	b.RunMakeInstall(outFile, vfOpt)
}

type MesonProject struct {
	builder *Builder
}

func (p *MesonProject) CoreBuilder() *Builder {
	return p.builder
}

func NewMesonProject(repo *RepoInfo, buildEnv *BuildEnv, libType LibType) Project {
	builder := NewBuilder(repo, buildEnv, libType)
	return &MesonProject{
		builder: builder,
	}
}

func (p *MesonProject) Init(opt *ProjectInitOptions) {
	if opt == nil {
		opt = &ProjectInitOptions{}
	}

	bp := p.builder
	bp.CloneAndGotoRepoSource()

	args := bp.GetMesonSetupArgsWithOptions(opt.GetMesonSetupArgsOptions)
	if len(opt.Args) > 0 {
		args = append(args, opt.Args...)
	}
	env := opt.Env

	var genOpt *RunMesonSetupOptions
	if opt.RunMesonSetupOptions != nil {
		genOpt = opt.RunMesonSetupOptions
		genOpt.Args = append(args, genOpt.Args...)
		genOpt.Env = append(env, genOpt.Env...)
	} else {
		genOpt = &RunMesonSetupOptions{
			Args: args,
			Env:  env,
		}
	}

	bp.RunMesonSetup(genOpt)
}

func (p *MesonProject) Build() {
	b := p.builder
	b.GoToBuildDir()
	b.RunMesonCompile()
}

func (p *MesonProject) Install(outFile string, vfOpt *VerifyFileOptions) {
	b := p.builder
	b.RunMesonInstall(outFile, vfOpt)
}
