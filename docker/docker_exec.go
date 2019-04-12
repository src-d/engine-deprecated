package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
)

func Exec(ctx context.Context, interactive bool, containerName string, args ...string) error {
	return exec(context.Background(), false, interactive, containerName, args...)
}

func ExecAndAttach(ctx context.Context, interactive bool, containerName string, args ...string) error {
	return exec(context.Background(), true, interactive, containerName, args...)
}

func exec(ctx context.Context, attach, interactive bool, containerName string, args ...string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return err
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
		return err
	}

	if attach {
		hjResp, err := c.ContainerExecAttach(ctx, idResp.ID, startCheck)
		if err != nil {
			return err
		}

		in, out, _ := term.StdStreams()
		if interactive {
			err = <-attachStdio(&hjResp, in, out, true)
		} else {
			err = <-attachStdout(&hjResp, out)
		}

		if err != nil {
			return err
		}
	}

	err = c.ContainerExecStart(ctx, idResp.ID, startCheck)
	if err != nil {
		return err
	}

	if interactive {
		container, err := Info(containerName)
		if err != nil {
			return err
		}

		monitorTtySize(c, container.ID)
	}

	return nil
}

func withRawTerminal(in io.ReadCloser, fn func() error) func() error {
	return func() error {
		fd, isTerminal := term.GetFdInfo(in)
		if isTerminal {
			var prevState *term.State
			// set terminal into raw mode to propagate special
			// characters
			prevState, err := term.SetRawTerminal(fd)
			if err != nil {
				return err
			}

			defer func() {
				err = term.RestoreTerminal(fd, prevState)
			}()
		}

		return fn()
	}
}

func attachStdin(resp *types.HijackedResponse, in io.ReadCloser, rawTerminal bool) chan error {
	inputDone := make(chan error)

	go func() {
		do := func() error {
			_, err := io.Copy(resp.Conn, in)
			if err != nil {
				return err
			}

			if err = resp.CloseWrite(); err != nil {
				logrus.Debugf("Couldn't send EOF: %s", err)
			}

			return err
		}

		if rawTerminal {
			do = withRawTerminal(in, do)
		}

		inputDone <- do()
	}()

	return inputDone
}

func attachStdout(resp *types.HijackedResponse, out io.Writer) chan error {
	outputDone := make(chan error)

	go func() {
		_, err := io.Copy(out, resp.Reader)
		outputDone <- err
		resp.CloseWrite()
	}()

	return outputDone
}

func attachStdio(resp *types.HijackedResponse, in io.ReadCloser, out io.Writer, rawTerminal bool) chan error {
	done := make(chan error)
	go func() {
		do := func() error {
			inputDone := attachStdin(resp, in, false)
			outputDone := attachStdout(resp, out)

			select {
			case err := <-outputDone:
				return err
			case err := <-inputDone:
				if err == nil {
					// Wait for output to complete streaming.
					err = <-outputDone
				}

				return err
			}
		}

		if rawTerminal {
			do = withRawTerminal(in, do)
		}

		done <- do()
	}()

	return done
}
