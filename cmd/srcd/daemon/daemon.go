package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

const (
	daemonImage  = "srcd-cli/daemon"
	daemonName   = "srcd-cli-daemon"
	daemonPort   = "4242"
	dockerSocket = "/var/run/docker.sock"
)

func DockerVersion() (string, error) { return docker.Version() }
func IsRunning() (bool, error)       { return docker.IsRunning(daemonName) }
func Kill() error                    { return docker.Kill(daemonName) }

func Client() (api.EngineClient, error) {
	info, err := docker.InfoOrStart(daemonName, Start)
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

func Start() error {
	logrus.Infof("starting srcd daemon")

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

	return docker.Start(ctx, config, host, daemonName)
}
