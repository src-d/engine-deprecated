package engine

import (
	"context"

	api "github.com/src-d/engine-cli/api"
)

var _ api.EngineServer = new(Server)

type Server struct {
	version string
	workdir string
}

func NewServer(version, workdir string) *Server {
	return &Server{
		version: version,
		workdir: workdir,
	}
}

func (s *Server) Version(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	return &api.VersionResponse{Version: s.version}, nil
}
