package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/components"
	"github.com/src-d/engine-cli/docker"
)

const (
	daemonImage  = "srcd-cli/daemon"
	daemonName   = "srcd-cli-daemon"
	daemonPort   = "4242"
	dockerSocket = "/var/run/docker.sock"
	workdirKey   = "WORKDIR"
)

func DockerVersion() (string, error) { return docker.Version() }
func IsRunning() (bool, error)       { return docker.IsRunning(daemonName) }

func Kill() error {
	cmps, err := components.List(context.Background(), components.IsWorkingDirDependant)
	if err != nil {
		return err
	}

	for _, cmp := range cmps {
		if err := docker.Kill(cmp); err != nil {
			return err
		}
	}

	return docker.Kill(daemonName)
}

// Client will return a new EngineClient to interact with the daemon. If the
// daemon is not started already, it will start it at the working directory.
func Client() (api.EngineClient, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	info, err := docker.InfoOrStart(
		daemonName,
		start(docker.WithEnv(workdirKey, wd)),
	)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("0.0.0.0:%d", info.Ports[0].PublicPort)
	// TODO(campoy): add security
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return api.NewEngineClient(conn), nil
}

func Start(workdir string) error {
	return start(docker.WithEnv(workdirKey, workdir))()
}

func start(opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image:        daemonImage,
			ExposedPorts: nat.PortSet{"4242": {}},
			Volumes:      map[string]struct{}{dockerSocket: {}},
		}

		host := &container.HostConfig{
			PortBindings: nat.PortMap{daemonPort: {{HostPort: "4242"}}},
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: dockerSocket,
				Target: dockerSocket,
			}},
		}

		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, daemonName)
	}
}
