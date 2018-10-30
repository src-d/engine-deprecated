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
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
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

	filter := filters.NewArgs()
	filter.Add("name", name)

	cs, err := c.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filter,
	})
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

func List() ([]Container, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	return c.ContainerList(context.Background(), types.ContainerListOptions{All: true})
}

func IsRunning(name string) (bool, error) {
	info, err := Info(name)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// apperantly there is no constant for it in API
	// use string value from documentation
	return info.State == "running", nil
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

func WithVolume(name, containerPath string) ConfigOption {
	return withVolume(mount.TypeVolume, name, containerPath)
}

func WithSharedDirectory(hostPath, containerPath string) ConfigOption {
	return withVolume(mount.TypeBind, hostPath, containerPath)
}

func withVolume(typ mount.Type, hostPath, containerPath string) ConfigOption {
	return func(cfg *container.Config, hc *container.HostConfig) {
		if cfg.Volumes == nil {
			cfg.Volumes = make(map[string]struct{})
		}

		cfg.Volumes[hostPath] = struct{}{}

		hc.Mounts = append(hc.Mounts, mount.Mount{
			Type:   typ,
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

// WithCmd appends arguments to the cmd arguments.
func WithCmd(args ...string) ConfigOption {
	return func(cfg *container.Config, hc *container.HostConfig) {
		cfg.Cmd = append(cfg.Cmd, args...)
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
		if err := c.ContainerRemove(ctx, res.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			logrus.Errorf("could not remove container after failing to create it")
		}
		return errors.Wrapf(err, "could not start container: %s", name)
	}

	// TODO: remove this hack
	time.Sleep(time.Second)

	err = connectToNetwork(ctx, res.ID)
	return errors.Wrapf(err, "could not connect to network")
}

func CreateVolume(ctx context.Context, name string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	_, err = c.VolumeInspect(ctx, name)
	if err == nil {
		return nil
	}

	_, err = c.VolumeCreate(ctx, volume.VolumesCreateBody{Name: name})
	return err
}

type Volume = types.Volume

func ListVolumes(ctx context.Context) ([]*Volume, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	list, err := c.VolumeList(ctx, filters.Args{})
	if err != nil {
		return nil, errors.Wrap(err, "could not get list of volumes")
	}

	return list.Volumes, nil
}

func RemoveVolume(ctx context.Context, id string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	return c.VolumeRemove(ctx, id, true)
}

func RemoveImage(ctx context.Context, id string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	_, err = c.ImageRemove(ctx, id, types.ImageRemoveOptions{Force: true})
	return err
}

const networkName = "srcd-cli-network"

func connectToNetwork(ctx context.Context, containerID string) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	if _, err := c.NetworkInspect(ctx, networkName); err != nil {
		logrus.Debugf("couldn't find network %s: %v", networkName, err)
		logrus.Infof("creating %s docker network", networkName)
		_, err = c.NetworkCreate(ctx, networkName, types.NetworkCreate{})
		if err != nil {
			return errors.Wrap(err, "could not create network")
		}
	}
	return c.NetworkConnect(ctx, networkName, containerID, nil)
}

func RemoveNetwork(ctx context.Context) error {
	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	resp, err := c.NetworkInspect(ctx, networkName)
	if client.IsErrNetworkNotFound(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "could not inspect network")
	}

	return c.NetworkRemove(ctx, resp.ID)
}

func GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	reader, err := c.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Since:      time.Now().Format(time.RFC3339Nano),
	})

	return reader, err
}
