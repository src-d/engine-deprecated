package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
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
			Name:         gitbaseWeb.Name,
			Start:        createGitbaseWeb(docker.WithPort(port, gitbaseWebPrivatePort)),
			Dependencies: []Component{s.gitbaseComponent()},
		})
	case bblfshWeb.Name:
		return Run(Component{
			Name:         bblfshWeb.Name,
			Start:        createBblfshWeb(docker.WithPort(port, bblfshWebPrivatePort)),
			Dependencies: []Component{s.bblfshComponent()},
		})
	case bblfshd.Name:
		return Run(s.bblfshComponent())
	case gitbase.Name:
		return Run(s.gitbaseComponent())
	default:
		return fmt.Errorf("can't start unknown component %s", name)
	}
}

func (s *Server) gitbaseComponent() Component {
	indexDir := join(s.datadir, "gitbase", s.workdirHash)

	return Component{
		Name: gitbase.Name,
		Start: createGitbase(
			docker.WithSharedDirectory(s.workdir, gitbaseMountPath),
			docker.WithSharedDirectory(indexDir, gitbaseIndexMountPath),
			docker.WithPort(gitbasePort, gitbasePort),
		),
		Dependencies: []Component{
			s.bblfshComponent(),
		},
	}
}

func (s *Server) bblfshComponent() Component {
	return Component{
		Name: bblfshd.Name,
		Start: createBbblfshd(
			s.installStableDrivers,
			docker.WithVolume(components.BblfshVolume, bblfshMountPath),
			docker.WithPort(bblfshParsePort, bblfshParsePort),
		),
	}
}

func inferSeparator(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "\\"
	}
	return "/"
}

// join the parts of a path using the separator of the detected OS.
func join(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	sep := inferSeparator(parts[0])

	for i, p := range parts {
		if i == 0 {
			parts[i] = strings.TrimRight(p, sep)
		} else {
			parts[i] = strings.Trim(p, sep)
		}
	}

	return strings.Join(parts, sep)
}
