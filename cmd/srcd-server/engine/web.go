package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/components"
	"github.com/src-d/engine-cli/docker"
)

const (
	gitbaseWebPrivatePort = 80
	bblfshWebPrivatePort  = 80
)

var (
	gitbaseWeb = components.GitbaseWeb
	bblfshWeb  = components.BblfshWeb
)

// StartBblfshWeb starts a bblfsh web.
func (s *Server) StartBblfshWeb(
	ctx context.Context,
	r *api.StartBblfshWebRequest,
) (*api.StartBblfshWebResponse, error) {
	if err := s.startBblfshWeb(ctx, int(r.Port)); err != nil {
		return nil, err
	}

	return &api.StartBblfshWebResponse{}, nil
}

func (s *Server) StopBblfshWeb(
	ctx context.Context,
	_ *api.StopBblfshWebRequest,
) (*api.StopBblfshWebResponse, error) {
	return &api.StopBblfshWebResponse{}, docker.Kill(bblfshWeb.Name)
}

func (s *Server) startBblfshWeb(
	ctx context.Context,
	port int,
) error {
	info, err := docker.Info(bblfshWeb.Name)
	if err != nil && err != docker.ErrNotFound {
		return err
	}

	if info != nil {
		for _, p := range info.Ports {
			if int(p.PublicPort) == port {
				return nil
			}
		}

		if err := docker.Kill(bblfshWeb.Name); err != nil {
			return err
		}
	}

	return Run(Component{
		Name:  bblfshWeb.Name,
		Start: createBblfshWeb(docker.WithPort(port, bblfshWebPrivatePort)),
		Dependencies: []Component{{
			Name:  bblfshd.Name,
			Start: createBbblfshd,
		}},
	})
}

// StartGitbaseWeb starts a gitbase web.
func (s *Server) StartGitbaseWeb(
	ctx context.Context,
	r *api.StartGitbaseWebRequest,
) (*api.StartGitbaseWebResponse, error) {
	if err := s.startGitbaseWeb(ctx, int(r.Port)); err != nil {
		return nil, err
	}

	return &api.StartGitbaseWebResponse{}, nil
}

func (s *Server) StopGitbaseWeb(
	ctx context.Context,
	_ *api.StopGitbaseWebRequest,
) (*api.StopGitbaseWebResponse, error) {
	return &api.StopGitbaseWebResponse{}, docker.Kill(gitbaseWeb.Name)
}

func (s *Server) startGitbaseWeb(
	ctx context.Context,
	port int,
) error {
	info, err := docker.Info(gitbaseWeb.Name)
	if err != nil && err != docker.ErrNotFound {
		return err
	}

	if info != nil {
		for _, p := range info.Ports {
			if int(p.PublicPort) == port {
				return nil
			}
		}

		if err := docker.Kill(gitbaseWeb.Name); err != nil {
			return err
		}
	}

	return Run(Component{
		Name:  gitbaseWeb.Name,
		Start: createGitbaseWeb(docker.WithPort(port, bblfshWebPrivatePort)),
		Dependencies: []Component{{
			Name:  gitbase.Name,
			Start: createGitbase(docker.WithVolume(s.workdir, gitbaseMountPath)),
			Dependencies: []Component{{
				Name:  bblfshd.Name,
				Start: createBbblfshd,
			}},
		}},
	})
}

func createBblfshWeb(opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		if err := docker.EnsureInstalled(bblfshWeb.Image, ""); err != nil {
			return err
		}

		logrus.Infof("starting bblfshd web")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config := &container.Config{
			Image: bblfshWeb.Image,
			Cmd:   []string{fmt.Sprintf("-bblfsh-addr=%s:%d", bblfshd.Name, bblfshParsePort)},
		}
		host := &container.HostConfig{
			// TODO(erizocosmico): Bblfsh web tries to connect to bblfsh before
			// we have a change to join to the network, so we have to link the two
			// containers.
			Links: []string{bblfshd.Name},
		}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, bblfshWeb.Name)
	}
}

func createGitbaseWeb(opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		if err := docker.EnsureInstalled(gitbaseWeb.Image, ""); err != nil {
			return err
		}

		logrus.Infof("starting gitbase web")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config := &container.Config{
			Image: gitbaseWeb.Image,
			Env: []string{
				fmt.Sprintf("GITBASEPG_DB_CONNECTION=root@tcp(%s)/none?maxAllowedPacket=4194304", gitbase.Name),
				fmt.Sprintf("GITBASEPG_BBLFSH_SERVER_URL=%s:%d", bblfshd.Name, bblfshParsePort),
			},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbaseWeb.Name)
	}
}
