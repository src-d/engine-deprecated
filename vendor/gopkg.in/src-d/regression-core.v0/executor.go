package regression

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Executor structure holds information and functionality to execute
// commands and get resource usage.
type Executor struct {
	command  string
	args     []string
	out      string
	Executed bool

	// metrics
	rusage *syscall.Rusage
	wall   time.Duration
}

// ErrNotRun means that the command was not started
var ErrNotRun = fmt.Errorf("command still was not executed")

// ErrRusageNotAvailable means that resource usage could not be collected
var ErrRusageNotAvailable = fmt.Errorf("rusage information not available")

// NewExecutor creates a new Executor struct.
func NewExecutor(command string, args ...string) (*Executor, error) {
	return &Executor{
		command: command,
		args:    args,
	}, nil
}

// Run executes the command and collects resource usage.
func (e *Executor) Run() error {
	defer func() { e.Executed = true }()

	cmd := exec.Command(e.command, e.args...)

	start := time.Now()

	out, err := cmd.CombinedOutput()
	e.out = string(out)
	if err != nil {
		return err
	}

	e.wall = time.Since(start)

	rusage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if ok {
		e.rusage = rusage
	}

	return nil
}

// Out retrieves stdout+stderr from the executed command.
func (e *Executor) Out() (string, error) {
	if !e.Executed {
		return "", ErrNotRun
	}

	return e.out, nil
}

// Rusage returns resource usage data.
func (e *Executor) Rusage() (*syscall.Rusage, error) {
	if !e.Executed {
		return nil, ErrNotRun
	}

	if e.rusage == nil {
		return nil, ErrRusageNotAvailable
	}

	return e.rusage, nil
}

// Wall returns time consumed by the execution.
func (e *Executor) Wall() (time.Duration, error) {
	if !e.Executed {
		return 0 * time.Second, ErrNotRun
	}

	return e.wall, nil
}
