package daemon

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	daemonPort   = "4242"
	dockerSocket = "/var/run/docker.sock"
	workdirKey   = "WORKDIR"
)

// cli version set by src-d command
var cliVersion = ""

// SetCliVersion sets cli version
func SetCliVersion(v string) {
	cliVersion = v
}

func DockerVersion() (string, error) { return docker.Version() }
func IsRunning() (bool, error)       { return docker.IsRunning(components.Daemon.Name, "") }

// Kill stops the daemon, and any of its dependencies. If it was not running it
// is ignored and does not produce an error
func Kill() error {
	cmps, err := components.List(
		context.Background(),
		true,
		components.IsWorkingDirDependant,
		components.IsRunningFilter)
	if err != nil {
		return err
	}

	for _, cmp := range cmps {
		logrus.Infof("removing container %s", cmp.Name)

		if err := cmp.Kill(); err != nil {
			return err
		}
	}

	return nil
}

// Client will return a new EngineClient to interact with the daemon. If the
// daemon is not started already, it will start it at the working directory.
func Client() (api.EngineClient, error) {
	info, err := ensureStarted()
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

func GetLogs() (io.ReadCloser, error) {
	info, err := ensureStarted()
	if err != nil {
		return nil, err
	}

	return docker.GetLogs(context.Background(), info.ID)
}

func ensureStarted() (*docker.Container, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return start(wd)
}

func start(workdir string) (*docker.Container, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get home dir")
	}

	tag, hasNew, err := docker.GetCompatibleTag(components.Daemon.Image, cliVersion)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get compatible daemon version")
	}

	if hasNew {
		logrus.Warn("new version of engine is available. Please download the latest release here: https://github.com/src-d/engine/releases")
	}

	datadir := filepath.Join(homedir, ".srcd")
	if err := setupDataDirectory(workdir, datadir); err != nil {
		return nil, err
	}

	if err := docker.EnsureInstalled(components.Daemon.Image, tag); err != nil {
		return nil, err
	}

	return docker.InfoOrStart(
		context.Background(),
		components.Daemon.Name,
		createDaemon(workdir, datadir, tag),
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

func createDaemon(workdir, datadir, tag string) docker.StartFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image:        fmt.Sprintf("%s:%s", components.Daemon.Image, tag),
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

		return docker.Start(ctx, config, host, components.Daemon.Name)
	}
}
