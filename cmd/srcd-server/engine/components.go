package engine

import (
	"context"
	"fmt"

	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

// Component to be run.
type Component struct {
	Name         string
	Start        docker.StartFunc
	Dependencies []Component
}

// Run the given components if they're not already running. It will recursively
// run all the component dependencies.
func Run(cs ...Component) error {
	return run(cs, make(map[string]struct{}))
}

func run(cs []Component, seen map[string]struct{}) error {
	for _, c := range cs {
		if len(c.Dependencies) > 0 {
			if err := run(c.Dependencies, seen); err != nil {
				return err
			}
		}

		if _, ok := seen[c.Name]; ok {
			continue
		}

		seen[c.Name] = struct{}{}
		_, err := docker.InfoOrStart(c.Name, c.Start)
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
	return &api.StartComponentResponse{}, s.startComponentAtPort(r.Name, int(r.Port))
}

func (s *Server) StopComponent(
	ctx context.Context,
	r *api.StopComponentRequest,
) (*api.StopComponentResponse, error) {
	return &api.StopComponentResponse{}, docker.Kill(r.Name)
}

func (s *Server) startComponent(name string) error {
	return s.startComponentAtPort(name, -1)
}

func (s *Server) startComponentAtPort(name string, port int) error {
	switch name {
	case gitbaseWeb.Name:
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
	case bblfshWeb.Name:
		return Run(Component{
			Name:  bblfshWeb.Name,
			Start: createBblfshWeb(docker.WithPort(port, bblfshWebPrivatePort)),
			Dependencies: []Component{{
				Name:  bblfshd.Name,
				Start: createBbblfshd,
			}},
		})
	case bblfshd.Name:
		return Run(Component{Name: bblfshd.Name, Start: createBbblfshd})
	case gitbase.Name:
		return Run(Component{
			Name: gitbase.Name,
			Start: createGitbase(
				docker.WithVolume(s.workdir, gitbaseMountPath),
			),
			Dependencies: []Component{{
				Name:  bblfshd.Name,
				Start: createBbblfshd,
			}},
		})
	default:
		return fmt.Errorf("can't start unknown component %s", name)
	}
}
