package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
	grpc "google.golang.org/grpc"
	"gopkg.in/src-d/go-log.v1"
)

const (
	dockerSocket = "/var/run/docker.sock"
	// maxMessageSize overrides default grpc max. message size to receive
	maxMessageSize = 100 * 1024 * 1024 // 100MB
	stateFileName  = ".state.json"
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
		log.Infof("removing container %s", cmp.Name)

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

	stateFile := filepath.Join(datadir, stateFileName)
	return os.RemoveAll(stateFile)
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

// startOptions is a configuration for src-d daemon
type startOptions struct {
	WorkDir string      `json:"workdir"`
	Config  *api.Config `json:"config"`
}

// Save persists configuration to a file
func (o *startOptions) Save() error {
	d, err := datadir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(d, 0755); err != nil {
		return errors.Wrapf(err, "can't create engine data directory")
	}

	f, err := os.Create(path.Join(d, stateFileName))
	if err != nil {
		return errors.Wrapf(err, "can't open state file for save")
	}
	defer f.Close()

	e := json.NewEncoder(f)
	return errors.Wrapf(e.Encode(o), "can't encode state into file")
}

func Start(workdir string) error {
	opts, err := saveState(workdir)
	if err != nil {
		return err
	}

	_, err = start(opts)
	return err
}

func saveState(workdir string) (startOptions, error) {
	cfg := config.File

	opts := startOptions{WorkDir: workdir, Config: cfg}
	if err := opts.Save(); err != nil {
		return startOptions{}, err
	}

	return opts, nil
}

func GetLogs() (io.ReadCloser, error) {
	info, err := ensureStarted()
	if err != nil {
		return nil, err
	}

	return docker.GetLogs(context.Background(), info.ID)
}

func ensureStarted() (*docker.Container, error) {
	running, err := docker.IsRunning(components.Daemon.Name, "")
	if err != nil {
		return nil, err
	}
	if running {
		return docker.Info(components.Daemon.Name)
	}

	d, err := datadir()
	if err != nil {
		return nil, err
	}

	statePath := path.Join(d, stateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		opts, err := saveState(wd)
		if err != nil {
			return nil, err
		}

		return start(opts)
	}

	f, err := os.Open(statePath)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open state file")
	}
	defer f.Close()

	var opts startOptions
	jd := json.NewDecoder(f)
	if err := jd.Decode(&opts); err != nil {
		return nil, errors.Wrapf(err, "can't decode state file")
	}

	return start(opts)
}

func start(opts startOptions) (*docker.Container, error) {
	return docker.InfoOrStart(
		context.Background(),
		components.Daemon.Name,
		createDaemon(opts),
	)
}

func createDaemon(opts startOptions) docker.StartFunc {
	workdir := filepath.ToSlash(opts.WorkDir)
	conf := opts.Config
	conf.SetDefaults()

	return func(ctx context.Context) error {
		cmp := components.Daemon
		hasNew, err := cmp.RetrieveVersion()
		if err != nil {
			log.Warningf("unable to list the available daemon versions on Docker Hub: ", err)
		}

		if hasNew {
			log.Warningf("new version of engine is available. Please download the latest release here: https://github.com/src-d/engine/releases")
		}

		if err := docker.EnsureInstalled(cmp.Image, cmp.Version); err != nil {
			return err
		}

		hostPort := strconv.Itoa(conf.Components.Daemon.Port)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		daemonPort := nat.Port(strconv.Itoa(components.DaemonPort))

		config := &container.Config{
			Image:        fmt.Sprintf("%s:%s", cmp.Image, cmp.Version),
			ExposedPorts: nat.PortSet{daemonPort: {}},
			Volumes:      map[string]struct{}{dockerSocket: {}},
			Cmd: []string{
				"serve",
				fmt.Sprintf("--workdir=%s", workdir),
				fmt.Sprintf("--host-os=%s", runtime.GOOS),
				fmt.Sprintf("--config=%s", conf.AsYaml()),
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
