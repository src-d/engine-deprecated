package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine-cli/api"
)

func DockerVersion() (string, error) {
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

const (
	daemonImage = "srcd-cli/daemon"
	daemonName  = "srcd-cli-daemon"
	daemonPort  = "4242"
)

var ErrNotFound = errors.New("container not found")

func IsRunning() (bool, error) {
	_, err := info()
	if err == ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func Client() (api.EngineClient, error) {
	info, err := info()
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("0.0.0.0:%d", info.Ports[0].PublicPort)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return api.NewEngineClient(conn), nil
}

func info() (*types.Container, error) {
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
		for _, name := range c.Names {
			if name[1:] == daemonName {
				return &c, nil
			}
		}
	}
	return nil, ErrNotFound
}

func Start() error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	config := &container.Config{Image: daemonImage, ExposedPorts: nat.PortSet{"4242": {}}}
	host := &container.HostConfig{PortBindings: nat.PortMap{daemonPort: {{HostPort: "4242"}}}}
	network := &network.NetworkingConfig{}

	res, err := c.ContainerCreate(ctx, config, host, network, daemonName)
	if err != nil {
		return errors.Wrapf(err, "could not create container %s", daemonName)
	}
	logrus.Debugf("info: %s", res)

	return c.ContainerStart(ctx, res.ID, types.ContainerStartOptions{})
}

func Kill() error {
	info, err := info()
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
