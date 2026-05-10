package ku

import (
	"github.com/mgenware/j9/v3"
)

type ProjectInitOptions struct {
	Args []string
	Env  []string
}

type Project interface {
	Init(opt *ProjectInitOptions)
	Build()
	Install(outFile []string)
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

func (p *CMakeProject) Init(opt *ProjectInitOptions) {
	if opt == nil {
		opt = &ProjectInitOptions{}
	}

	b := p.builder
	b.CloneAndGotoRepo()
	args := b.GetCmakeGenArgs()

	env := b.GetToolchainEnv(nil)
	b.RunCmakeGen(&RunCmakeGenOptions{
		Args: args,
		Env:  env,
	})
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
	b.CloneAndGotoRepo()

	env := b.GetToolchainEnv(&GetToolchainEnvOptions{
		MakeOnlySetCompilerFlags: true,
	})
	if len(opt.Env) > 0 {
		env = append(env, opt.Env...)
	}

	args := opt.Args
	b.RunMakeCleanRaw()
	b.Shell.Spawn(&j9.SpawnOpt{
		Name: "./configure",
		Args: args,
		Env:  env,
	})
}

func (p *MakeProject) Build() {
	b := p.builder
	b.RunMake()
}

func (p *MakeProject) Install(outFile []string) {
	b := p.builder
	b.RunMakeInstall(outFile)
}

type MesonProject struct {
	builder *Builder
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

	args := bp.GetMesonSetupArgs()
	if len(opt.Args) > 0 {
		args = append(args, opt.Args...)
	}
	bp.RunMesonSetup(&RunMesonSetupOptions{
		Args: args,
		Env:  opt.Env,
	})
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
