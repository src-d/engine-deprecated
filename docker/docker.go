package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

func Version() (string, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return "", errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ping, err := c.Ping(ctx)
	if err != nil {
		return "", errors.Wrap(err, "could not ping docker")
	}

	return ping.APIVersion, nil
}

var ErrNotFound = errors.New("container not found")

type Container = types.Container

func Info(name string) (*Container, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cs, err := c.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list containers")
	}

	for _, c := range cs {
		for _, n := range c.Names {
			if name == n[1:] {
				return &c, nil
			}
		}
	}
	return nil, ErrNotFound
}

func IsRunning(name string) (bool, error) {
	_, err := Info(name)
	if err == ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func Kill(name string) error {
	info, err := Info(name)
	if err != nil {
		return err
	}

	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return c.ContainerRemove(ctx, info.ID, types.ContainerRemoveOptions{Force: true})
}

func InfoOrStart(name string, start func() error) (*Container, error) {
	i, err := Info(name)
	if err == nil {
		return i, nil
	}
	if err := start(); err != nil {
		return nil, errors.Wrapf(err, "could not create %s", name)
	}
	return Info(name)
}
