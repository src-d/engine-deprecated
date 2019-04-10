package regression

import (
	"fmt"
)

type GitServer struct {
	*Server
	config Config
}

func NewGitServer(config Config) *GitServer {
	return &GitServer{
		Server: NewServer(),
		config: config,
	}
}

func (s *GitServer) Start() error {
	basePath := fmt.Sprintf("--base-path=%s", s.config.RepositoriesCache)
	port := fmt.Sprintf("--port=%d", s.config.GitServerPort)

	arg := []string{"daemon", basePath, port,
		"--export-all", s.config.RepositoriesCache}

	return s.Server.Start("git", arg...)
}

func (s *GitServer) Url(name string) string {
	return fmt.Sprintf("git://localhost:%d/%s", s.config.GitServerPort, name)
}
