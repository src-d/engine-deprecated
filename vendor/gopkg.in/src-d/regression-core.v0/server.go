package regression

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Server struct describes a daemon.
type Server struct {
	cmd *exec.Cmd
}

// NewServer creates a new Server struct.
func NewServer() *Server {
	return new(Server)
}

// Start executes a command in background.
func (s *Server) Start(name string, arg ...string) error {
	s.cmd = exec.Command(name, arg...)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	err := s.cmd.Start()

	// TODO: check that the server is ready (read stdout?)
	time.Sleep(1 * time.Second)

	return err
}

// Stop kill the daemon.
func (s *Server) Stop() error {
	err := syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
	if err != nil {
		return err
	}

	time.AfterFunc(3*time.Second, func() {
		if s.Alive() {
			s.cmd.Process.Signal(syscall.SIGKILL)
		}
	})

	_ = s.cmd.Wait()
	return nil
}

// Alive checks if the process is still running.
func (s *Server) Alive() bool {
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}

	err := s.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Rusage returns usage counters.
func (s *Server) Rusage() *syscall.Rusage {
	rusage, _ := s.cmd.ProcessState.SysUsage().(*syscall.Rusage)
	return rusage
}
