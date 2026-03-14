package ku

import (
	"os"
	"strings"

	"github.com/mgenware/j9/v3"
)

type Shell struct {
	Tunnel *j9.Tunnel
}

func NewShell(tunnel *j9.Tunnel) *Shell {
	return &Shell{Tunnel: tunnel}
}

func (s *Shell) Shell(cmd string) string {
	output := s.Tunnel.Shell(&j9.ShellOpt{
		Cmd: cmd})
	return strings.TrimSpace(string(output))
}

func (s *Shell) Spawn(opt *j9.SpawnOpt) {
	s.Tunnel.Spawn(opt)
}

func (s *Shell) SpawnRaw(opt *j9.SpawnOpt) error {
	return s.Tunnel.SpawnRaw(opt)
}

func (s *Shell) Logger() j9.Logger {
	return s.Tunnel.Logger()
}

func (s *Shell) CD(dir string) {
	s.Tunnel.CD(dir)
}

func (s *Shell) Quit(msg string) {
	s.Tunnel.Logger().Log(j9.LogLevelError, msg)
	os.Exit(1)
}
