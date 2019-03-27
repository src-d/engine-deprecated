package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	startComponentTimeout = 60 * time.Second
)

// Component to be run.
type Component struct {
	Name         string
	Start        docker.StartFunc
	Dependencies []Component
}

// Run the given components if they're not already running. It will recursively
// run all the component dependencies.
func Run(ctx context.Context, cs ...Component) error {
	return run(ctx, cs, make(map[string]struct{}))
}

func run(ctx context.Context, cs []Component, seen map[string]struct{}) error {
	for _, c := range cs {
		if len(c.Dependencies) > 0 {
			if err := run(ctx, c.Dependencies, seen); err != nil {
				return err
			}
		}

		if _, ok := seen[c.Name]; ok {
			continue
		}

		seen[c.Name] = struct{}{}
		_, err := docker.InfoOrStart(ctx, c.Name, c.Start)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) StartComponent(
	ctx context.Context,
	r *api.StartComponentRequest,
) (*api.StartComponentResponse, error) {
	port, err := s.startComponentAtPort(ctx, r.Name, int(r.Port))
	return &api.StartComponentResponse{Port: int32(port)}, err
}

func (s *Server) StopComponent(
	ctx context.Context,
	r *api.StopComponentRequest,
) (*api.StopComponentResponse, error) {
	return &api.StopComponentResponse{}, docker.RemoveContainer(r.Name)
}

func (s *Server) startComponent(ctx context.Context, name string) error {
	_, err := s.startComponentAtPort(ctx, name, 0)
	return err
}

// startComponentAtPort starts the container with the given public port binding.
// If port is 0, the one set in the initial --config will be used.
// If port is -1, the public port will be the same as the private one.
func (s *Server) startComponentAtPort(
	ctx context.Context, name string, port int,
) (int, error) {

	var err error
	publicPort := s.getPublicPort(name, port)

	switch name {
	case gitbaseWeb.Name:
		gbComp, err := s.gitbaseComponent(0)
		if err != nil {
			break
		}

		return publicPort, Run(ctx, Component{
			Name:         gitbaseWeb.Name,
			Start:        createGitbaseWeb(docker.WithPort(publicPort, components.GitbaseWebPort)),
			Dependencies: []Component{*gbComp},
		})
	case bblfshWeb.Name:
		bbfComp, err := s.bblfshComponent(0)
		if err != nil {
			break
		}

		return publicPort, Run(ctx, Component{
			Name:         bblfshWeb.Name,
			Start:        createBblfshWeb(docker.WithPort(publicPort, components.BblfshWebPort)),
			Dependencies: []Component{*bbfComp},
		})
	case bblfshd.Name:
		bbfComp, err := s.bblfshComponent(port)
		if err != nil {
			break
		}

		return publicPort, Run(ctx, *bbfComp)
	case gitbase.Name:
		gbComp, err := s.gitbaseComponent(port)
		if err != nil {
			break
		}

		return publicPort, Run(ctx, *gbComp)
	default:
		return 0, fmt.Errorf("can't start unknown component %s", name)
	}

	return 0, errors.Wrapf(err, "can't start component %s", name)
}

func (s *Server) getPublicPort(name string, requestedPort int) int {
	var defaultPort, privatePort int

	switch name {
	case gitbaseWeb.Name:
		defaultPort = s.config.Components.GitbaseWeb.Port
		privatePort = components.GitbaseWebPort
	case bblfshWeb.Name:
		defaultPort = s.config.Components.BblfshWeb.Port
		privatePort = components.BblfshWebPort
	case bblfshd.Name:
		defaultPort = s.config.Components.Bblfshd.Port
		privatePort = components.BblfshParsePort
	case gitbase.Name:
		defaultPort = s.config.Components.Gitbase.Port
		privatePort = components.GitbasePort
	}

	switch requestedPort {
	case 0:
		return defaultPort
	case -1:
		return privatePort
	default:
		return requestedPort
	}
}

func (s *Server) gitbaseComponent(port int) (*Component, error) {
	port = s.getPublicPort(gitbase.Name, port)

	indexDir := filepath.Join(s.datadir, "gitbase", s.workdirHash)

	workdirHostPath, err := docker.HostPath(s.workdir)
	if err != nil {
		return nil, errors.Wrapf(err, "can't process host path for workdir %s", s.workdir)
	}

	indexDirHostPath, err := docker.HostPath(indexDir)
	if err != nil {
		return nil, errors.Wrapf(err, "can't process host path for indexdir %s", indexDir)
	}

	bblfshComponent, err := s.bblfshComponent(0)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create %s component", bblfshd.Name)
	}

	return &Component{
		Name: gitbase.Name,
		Start: createGitbase(
			docker.WithSharedDirectory(workdirHostPath, gitbaseMountPath, s.hostOS),
			docker.WithSharedDirectory(indexDirHostPath, gitbaseIndexMountPath, s.hostOS),
			docker.WithPort(port, components.GitbasePort),
		),
		Dependencies: []Component{*bblfshComponent},
	}, nil
}

func (s *Server) bblfshComponent(port int) (*Component, error) {
	port = s.getPublicPort(bblfshd.Name, port)

	return &Component{
		Name: bblfshd.Name,
		Start: createBbblfshd(
			docker.WithPort(port, components.BblfshParsePort),
		),
	}, nil
}
