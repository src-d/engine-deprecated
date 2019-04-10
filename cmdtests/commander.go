// +build integration regression

package cmdtests

import (
	"path"
	"runtime"
	"time"

	"gotest.tools/icmd"
)

type Commander struct {
	bin string
}

func NewCommander(bin string) *Commander {
	return &Commander{bin: bin}
}

func (s *Commander) Bin() string {
	return s.bin
}

func (s *Commander) RunCmd(cmd string, args []string, cmdOperators ...icmd.CmdOp) *icmd.Result {
	args = append([]string{cmd}, args...)
	return icmd.RunCmd(icmd.Command(s.bin, args...), cmdOperators...)
}

func (s *Commander) RunCommand(cmd string, args ...string) *icmd.Result {
	return s.RunCmd(cmd, args)
}

func (s *Commander) StartCommand(cmd string, args []string, cmdOperators ...icmd.CmdOp) *icmd.Result {
	args = append([]string{cmd}, args...)
	return icmd.StartCmd(icmd.Command(s.bin, args...))
}

func (s *Commander) Wait(timeout time.Duration, r *icmd.Result) *icmd.Result {
	return icmd.WaitOnCmd(timeout, r)
}

// RunInit runs srcd init with workdir and custom config for integration tests
func (s *Commander) RunInit(workdir string) *icmd.Result {
	return s.RunInitWithTimeout(workdir, 0)
}

// RunInitWithTimeout runs srcd init with workdir and custom config for integration tests with timeout
func (s *Commander) RunInitWithTimeout(workdir string, timeout time.Duration) *icmd.Result {
	_, filename, _, _ := runtime.Caller(0)
	configFile := path.Join(path.Dir(filename), "..", "integration-testing-config.yaml")
	return s.RunCmd("init", []string{workdir, "--config", configFile}, icmd.WithTimeout(timeout))
}
