package daemon

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/config"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

const (
	dockerSocket = "/var/run/docker.sock"
	// maxMessageSize overrides default grpc max. message size to receive
	maxMessageSize = 100 * 1024 * 1024 // 100MB
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
		components.IsRunning)
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

// CleanUp removes all resources created by daemon on host
func CleanUp() error {
	datadir, err := datadir()
	if err != nil {
		return err
	}

	gitbaseIndexDir := filepath.Join(datadir, "gitbase")

	return os.RemoveAll(gitbaseIndexDir)
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
	conn, err := grpc.Dial(addr,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
		),
		grpc.WithInsecure())
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
	return docker.InfoOrStart(
		context.Background(),
		components.Daemon.Name,
		createDaemon(workdir),
	)
}

func setupDataDirectory(workdir, datadir string) error {
	hash := sha1.Sum([]byte(workdir))
	workdirHash := hex.EncodeToString(hash[:])

	paths := [][]string{
		[]string{datadir, "gitbase", workdirHash},
	}

	for _, path := range paths {
		if err := os.MkdirAll(filepath.ToSlash(filepath.Join(path...)), 0755); err != nil {
			return errors.Wrap(err, "unable to create data directory")
		}
	}

	return nil
}

func createDaemon(workdir string) docker.StartFunc {
	workdir = filepath.ToSlash(workdir)

	return func(ctx context.Context) error {
		cmp := components.Daemon
		hasNew, err := cmp.RetrieveVersion()
		if err != nil {
			logrus.Warn("unable to list the available daemon versions on Docker Hub: ", err)
		}

		if hasNew {
			logrus.Warn("new version of engine is available. Please download the latest release here: https://github.com/src-d/engine/releases")
		}

		datadir, err := datadir()
		if err != nil {
			return err
		}
		// we run the command inside docker so slashes must be always converted to unix-style
		datadir = filepath.ToSlash(datadir)

		if err := setupDataDirectory(workdir, datadir); err != nil {
			return err
		}

		if err := docker.EnsureInstalled(cmp.Image, cmp.Version); err != nil {
			return err
		}

		conf, err := config.Config()
		if err != nil {
			return err
		}

		conf.SetDefaults()
		hostPort := strconv.Itoa(conf.Components.Daemon.Port)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		daemonPort := nat.Port(strconv.Itoa(components.DaemonPort))

		config := &container.Config{
			Image:        fmt.Sprintf("%s:%s", cmp.Image, cmp.Version),
			ExposedPorts: nat.PortSet{daemonPort: {}},
			Volumes:      map[string]struct{}{dockerSocket: {}},
			Cmd: []string{
				fmt.Sprintf("--workdir=%s", workdir),
				fmt.Sprintf("--data=%s", datadir),
				fmt.Sprintf("--config=%s", config.YamlStringConfig()),
			},
		}

		host := &container.HostConfig{
			PortBindings: nat.PortMap{daemonPort: {{HostPort: hostPort}}},
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: dockerSocket,
				Target: dockerSocket,
			}},
		}

		return docker.Start(ctx, config, host, cmp.Name)
	}
}

func datadir() (string, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "unable to get home dir")
	}

	return filepath.Join(homedir, ".srcd"), nil
}
