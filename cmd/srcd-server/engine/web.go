package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

const (
	gitbaseWebName        = "srcd-cli-gitbase-web"
	gitbaseWebImage       = "srcd/gitbase-web"
	gitbaseWebPrivatePort = 80

	bblfshWebName        = "srcd-cli-bblfsh-web"
	bblfshWebImage       = "bblfsh/web"
	bblfshWebPrivatePort = 80
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
	return &api.StopBblfshWebResponse{}, docker.Kill(bblfshWebName)
}

func (s *Server) startBblfshWeb(
	ctx context.Context,
	port int,
) error {
	info, err := docker.Info(bblfshWebName)
	if err != nil && err != docker.ErrNotFound {
		return err
	}

	if info != nil {
		for _, p := range info.Ports {
			if int(p.PublicPort) == port {
				return nil
			}
		}

		if err := docker.Kill(bblfshWebName); err != nil {
			return err
		}
	}

	return Run(Component{
		Name:  bblfshWebName,
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
	return &api.StopGitbaseWebResponse{}, docker.Kill(gitbaseWebName)
}

func (s *Server) startGitbaseWeb(
	ctx context.Context,
	port int,
) error {
	info, err := docker.Info(gitbaseWebName)
	if err != nil && err != docker.ErrNotFound {
		return err
	}

	if info != nil {
		for _, p := range info.Ports {
			if int(p.PublicPort) == port {
				return nil
			}
		}

		if err := docker.Kill(gitbaseWebName); err != nil {
			return err
		}
	}

	return Run(Component{
		Name:  gitbaseWebName,
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
		if err := docker.EnsureInstalled(bblfshWebImage, ""); err != nil {
			return err
		}

		logrus.Infof("starting bblfshd web")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config := &container.Config{
			Image: bblfshWebImage,
			Cmd:   []string{fmt.Sprintf("-bblfsh-addr=%s:%d", bblfshd.Name, bblfshParsePort)},
		}
		host := &container.HostConfig{
			// TODO(erizocosmico): Bblfsh web tries to connect to bblfsh before
			// we have a change to join to the network, so we have to link the two
			// containers.
			Links: []string{bblfshd.Name},
		}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, bblfshWebName)
	}
}

func createGitbaseWeb(opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		if err := docker.EnsureInstalled(gitbaseWebImage, ""); err != nil {
			return err
		}

		logrus.Infof("starting gitbase web")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		config := &container.Config{
			Image: gitbaseWebImage,
			Env: []string{
				fmt.Sprintf("GITBASEPG_DB_CONNECTION=root@tcp(%s)/none?maxAllowedPacket=4194304", gitbase.Name),
				fmt.Sprintf("GITBASEPG_BBLFSH_SERVER_URL=%s:%d", bblfshd.Name, bblfshParsePort),
			},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbaseWebName)
	}
}
