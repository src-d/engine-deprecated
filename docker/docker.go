package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// IsInstalled checks whether an image is installed or not. If version is
// empty, it will check that any version is installed, otherwise it will check
// that the given version is installed.
func IsInstalled(ctx context.Context, image, version string) (bool, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return false, errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	imgs, err := c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "could not list images")
	}

	for _, i := range imgs {
		if len(i.RepoTags) == 0 {
			continue
		}

		if version == "" {
			img := strings.Split(i.RepoTags[0], ":")[0]
			if img == image {
				return true, nil
			}
		} else {
			id := image + ":" + version
			if id == i.RepoTags[0] {
				return true, nil
			}
		}
	}

	return false, nil
}

// Pull an image from docker hub with a specific version.
func Pull(ctx context.Context, image, version string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	id := image + ":" + version
	rc, err := c.ImagePull(ctx, id, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not pull image %q", id))
	}

	io.Copy(ioutil.Discard, rc)

	return rc.Close()
}

// EnsureInstalled checks whether an image is installed or not. If version is
// empty, it will check that any version is installed, otherwise it will check
// that the given version is installed. If the image is not installed, it will
// be automatically installed.
func EnsureInstalled(image, version string) error {
	ok, err := IsInstalled(context.Background(), image, version)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if version == "" {
		version = "latest"
	}
	id := image + ":" + version

	logrus.Infof("installing %q", id)

	if err := Pull(context.Background(), image, version); err != nil {
		return err
	}

	logrus.Infof("installed %q", id)

	return nil
}

type ConfigOption func(*container.Config, *container.HostConfig)

func WithEnv(key, value string) ConfigOption {
	return func(cfg *container.Config, hc *container.HostConfig) {
		cfg.Env = append(cfg.Env, key+"="+value)
	}
}

func WithVolume(hostPath, containerPath string) ConfigOption {
	return func(cfg *container.Config, hc *container.HostConfig) {
		if cfg.Volumes == nil {
			cfg.Volumes = make(map[string]struct{})
		}

		cfg.Volumes[hostPath] = struct{}{}

		hc.Mounts = append(hc.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostPath,
			Target: containerPath,
		})
	}
}

func WithPort(publicPort, privatePort int) ConfigOption {
	return func(cfg *container.Config, hc *container.HostConfig) {
		if cfg.ExposedPorts == nil {
			cfg.ExposedPorts = make(nat.PortSet)
		}

		if hc.PortBindings == nil {
			hc.PortBindings = make(nat.PortMap)
		}

		port := nat.Port(fmt.Sprint(privatePort))
		cfg.ExposedPorts[port] = struct{}{}
		hc.PortBindings[port] = append(
			hc.PortBindings[port],
			nat.PortBinding{HostPort: fmt.Sprint(publicPort)},
		)
	}
}

func ApplyOptions(c *container.Config, hc *container.HostConfig, opts ...ConfigOption) {
	for _, o := range opts {
		o(c, hc)
	}
}

type StartFunc func() error

func InfoOrStart(name string, start StartFunc) (*Container, error) {
	i, err := Info(name)
	if err == nil {
		return i, nil
	}

	if err := start(); err != nil {
		return nil, errors.Wrapf(err, "could not create %s", name)
	}

	return Info(name)
}

func Start(ctx context.Context, config *container.Config, host *container.HostConfig, name string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	res, err := c.ContainerCreate(ctx, config, host, &network.NetworkingConfig{}, name)
	if err != nil {
		return errors.Wrapf(err, "could not create container %s", name)
	}

	if err := c.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "could not start container: %s", name)
	}

	// TODO: remove this hack
	time.Sleep(time.Second)

	err = connectToNetwork(ctx, res.ID)
	return errors.Wrapf(err, "could not connect to network")
}

func connectToNetwork(ctx context.Context, containerID string) error {
	const networkName = "srcd-cli-network"

	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	if _, err := c.NetworkInspect(ctx, networkName); err != nil {
		logrus.Infof("couldn't find network %s: %v", networkName, err)
		logrus.Infof("creating it now")
		_, err = c.NetworkCreate(ctx, networkName, types.NetworkCreate{})
		if err != nil {
			return errors.Wrap(err, "could not create network")
		}
	}
	return c.NetworkConnect(ctx, networkName, containerID, nil)
}
