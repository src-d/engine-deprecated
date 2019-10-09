package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
)

type ExecInspect = types.ContainerExecInspect

func Exec(ctx context.Context, interactive bool, containerName string, args ...string) (*ExecInspect, error) {
	return exec(context.Background(), false, interactive, containerName, args...)
}

func ExecAndAttach(ctx context.Context, interactive bool, containerName string, args ...string) (*ExecInspect, error) {
	return exec(context.Background(), true, interactive, containerName, args...)
}

func exec(ctx context.Context, attach, interactive bool, containerName string, args ...string) (*ExecInspect, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	var config types.ExecConfig
	var startCheck types.ExecStartCheck
	if attach {
		config = types.ExecConfig{
			Tty:          true,
			AttachStdin:  true,
			AttachStderr: true,
			AttachStdout: true,
			Cmd:          args,
		}
		startCheck = types.ExecStartCheck{Tty: true}
	} else {
		config = types.ExecConfig{Cmd: args}
		startCheck = types.ExecStartCheck{}
	}

	idResp, err := c.ContainerExecCreate(ctx, containerName, config)
	if err != nil {
		return nil, err
	}

	if attach {
		hjResp, err := c.ContainerExecAttach(ctx, idResp.ID, startCheck)
		if err != nil {
			return nil, err
		}

		in, out, _ := term.StdStreams()
		sa := stdioAttacher{resp: &hjResp, in: in, out: out}

		var fn attachFn
		if interactive {
			container, err := Info(containerName)
			if err != nil {
				return nil, err
			}

			monitorTtySize(c, container.ID)
			fn = sa.attachStdio
		} else {
			fn = sa.attachStdout
		}

		if err = <-sa.withRawTerminal(fn); err != nil {
			return nil, err
		}
	} else {
		err = c.ContainerExecStart(ctx, idResp.ID, startCheck)
		if err != nil {
			return nil, err
		}
	}

	insResp, err := c.ContainerExecInspect(ctx, idResp.ID)
	if err != nil {
		return nil, err
	}

	return &insResp, nil
}

type attachFn func(chan<- error)

type stdioAttacher struct {
	resp *types.HijackedResponse
	in   io.ReadCloser
	out  io.Writer
}

func (sa *stdioAttacher) attachStdin(done chan<- error) {
	go func() {
		do := func() error {
			_, err := io.Copy(sa.resp.Conn, sa.in)
			if err != nil {
				return err
			}

			if err = sa.resp.CloseWrite(); err != nil {
				logrus.Debugf("Couldn't send EOF: %s", err)
			}

			return err
		}

		done <- do()
	}()
}

func (sa *stdioAttacher) attachStdout(done chan<- error) {
	go func() {
		_, err := io.Copy(sa.out, sa.resp.Reader)
		done <- err
		sa.resp.CloseWrite()
	}()
}

func (sa *stdioAttacher) attachStdio(done chan<- error) {
	go func() {
		inputDone := make(chan error)
		outputDone := make(chan error)

		sa.attachStdin(inputDone)
		sa.attachStdout(outputDone)

		select {
		case err := <-outputDone:
			done <- err
		case err := <-inputDone:
			if err == nil {
				// Wait for output to complete streaming.
				err = <-outputDone
			}

			done <- err
		}
	}()
}

func (sa *stdioAttacher) withRawTerminal(fn attachFn) <-chan error {
	fd, isTerminal := term.GetFdInfo(sa.in)
	var err error
	var prevState *term.State
	done := make(chan error)
	wrappedDone := make(chan error)
	if isTerminal {
		// set terminal into raw mode to propagate special
		// characters
		prevState, err = term.SetRawTerminal(fd)
		if err != nil {
			go func() {
				done <- err
			}()

			return done
		}
	}

	go func() {
		wrappedErr := <-wrappedDone
		if isTerminal {
			if err = term.RestoreTerminal(fd, prevState); err != nil {
				done <- err
				return
			}
		}

		done <- wrappedErr
	}()

	fn(wrappedDone)
	return done
}
