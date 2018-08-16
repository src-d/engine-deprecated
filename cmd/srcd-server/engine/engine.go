package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	bblfsh "gopkg.in/bblfsh/client-go.v2"
	enry "gopkg.in/src-d/enry.v1"

	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

var _ api.EngineServer = new(Server)

type Server struct {
	version string
}

func NewServer(version string) *Server {
	return &Server{
		version: version,
	}
}

func (s *Server) Version(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	return &api.VersionResponse{Version: s.version}, nil
}

const bblfshdName = "srcd-cli-bblfshd"

func (s *Server) Parse(ctx context.Context, req *api.ParseRequest) (*api.ParseResponse, error) {
	logrus.Infof("got parse request")
	lang := req.Lang
	if lang == "" {
		lang = enry.GetLanguage(req.Name, req.Content)
	}
	if req.Kind == api.ParseRequest_LANG {
		return &api.ParseResponse{Lang: lang}, nil
	}

	// check whether bblfshd is installed or not

	info, err := docker.InfoOrStart(bblfshdName, createBbblfshd)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", info.Ports[0].IP, info.Ports[0].PublicPort)
	logrus.Infof("connecting to bblfshd on %s", addr)
	client, err := bblfsh.NewClient(addr)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh")
	}

	res, err := client.NewParseRequest().
		Language(lang).
		Content(string(req.Content)).
		Filename(req.Name).
		DoWithContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse")
	}

	j, _ := json.MarshalIndent(res.UAST, "", "\t")
	logrus.Debugf("uast: %s", j)

	return &api.ParseResponse{Lang: "works!"}, nil
}

func createBbblfshd() error {
	logrus.Infof("starting bblfshd daemon")

	c, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "could not create docker client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &container.Config{
		Image:        "bblfsh/bblfshd",
		ExposedPorts: nat.PortSet{"9432": {}},
	}
	host := &container.HostConfig{
		Privileged:   true,
		PortBindings: nat.PortMap{"9432": {{HostPort: "9432"}}},
		// TODO: add volume to store drivers
	}
	network := &network.NetworkingConfig{}

	res, err := c.ContainerCreate(ctx, config, host, network, bblfshdName)
	if err != nil {
		return errors.Wrapf(err, "could not create container %s", bblfshdName)
	}

	if err := c.ContainerStart(ctx, res.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrapf(err, "could not start container: %s", bblfshdName)
	}

	// TODO(campoy): wait for gRPC server to be actually running.
	time.Sleep(1000 * time.Millisecond)
	return nil
}
