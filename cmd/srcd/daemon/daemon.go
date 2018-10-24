package daemon

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	daemonImage  = "srcd/cli-daemon"
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

	info, err := start(wd)
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
	_, err := start(workdir)
	return err
}

func start(workdir string) (*docker.Container, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get home dir")
	}

	datadir := filepath.Join(homedir, ".srcd")
	if err := setupDataDirectory(workdir, datadir); err != nil {
		return nil, err
	}

	if err := docker.EnsureInstalled(daemonImage, ""); err != nil {
		return nil, err
	}

	return docker.InfoOrStart(
		context.Background(),
		daemonName,
		createDaemon(workdir, datadir),
	)
}

func setupDataDirectory(workdir, datadir string) error {
	hash := sha1.Sum([]byte(workdir))
	workdirHash := hex.EncodeToString(hash[:])

	paths := [][]string{
		[]string{datadir, "gitbase", workdirHash},
	}

	for _, path := range paths {
		if err := os.MkdirAll(filepath.Join(path...), 0755); err != nil {
			return errors.Wrap(err, "unable to create data directory")
		}
	}

	return nil
}

func createDaemon(workdir, datadir string) docker.StartFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image:        daemonImage,
			ExposedPorts: nat.PortSet{"4242": {}},
			Volumes:      map[string]struct{}{dockerSocket: {}},
			Cmd: []string{
				fmt.Sprintf("--workdir=%s", workdir),
				fmt.Sprintf("--data=%s", datadir),
			},
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
}
