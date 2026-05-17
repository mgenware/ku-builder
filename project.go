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
	Init(opt *ProjectInitOptions)
	Build()
	Install(outFile []string)

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
	b.CloneAndGotoRepo()

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

func (p *CMakeProject) Install(outFile []string) {
	b := p.builder
	b.RunCmakeInstall(outFile)
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
	repoDir := b.CloneAndGotoRepo()

	env := b.GetMakeToolchainEnv(&GetToolchainEnvOptions{
		MakeOnlySetCompilerFlags:  true,
		MakeOnlyExtraCAndCXXFlags: opt.MakeExtraCAndCXXFlags,
		MakeOnlyExtraLDFlags:      opt.MakeExtraLDFlags,
	})
	if len(opt.Env) > 0 {
		env = append(env, opt.Env...)
	}

	// Run ./configure at build dir, not repo dir.
	b.GoToBuildDir()
	configureFilePath := filepath.Join(repoDir, "configure")
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

func (p *MakeProject) Install(outFile []string) {
	b := p.builder
	b.RunMakeInstall(outFile)
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
	bp.CloneAndGotoRepo()

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

func (p *MesonProject) Install(outFile []string) {
	b := p.builder
	b.RunMesonInstall(outFile)
}
