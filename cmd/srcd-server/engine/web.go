package engine

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
	"gopkg.in/src-d/go-log.v1"
)

const gitbaseWebSelectLimit = 0

var (
	gitbaseWeb = components.GitbaseWeb
	bblfshWeb  = components.BblfshWeb
)

func createBblfshWeb(opts ...docker.ConfigOption) docker.StartFunc {
	return func(ctx context.Context) error {
		if err := docker.EnsureInstalled(bblfshWeb.Image, bblfshWeb.Version); err != nil {
			return err
		}

		log.Infof("starting bblfsh web")

		ctx, cancel := context.WithTimeout(context.Background(), startComponentTimeout)
		defer cancel()

		config := &container.Config{
			Image: bblfshWeb.ImageWithVersion(),
			Cmd:   []string{fmt.Sprintf("-bblfsh-addr=%s:%d", bblfshd.Name, components.BblfshParsePort)},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, bblfshWeb.Name)
	}
}

func createGitbaseWeb(opts ...docker.ConfigOption) docker.StartFunc {
	return func(ctx context.Context) error {
		if err := docker.EnsureInstalled(gitbaseWeb.Image, gitbaseWeb.Version); err != nil {
			return err
		}

		log.Infof("starting gitbase web")

		ctx, cancel := context.WithTimeout(context.Background(), startComponentTimeout)
		defer cancel()

		config := &container.Config{
			Image: gitbaseWeb.ImageWithVersion(),
			Env: []string{
				fmt.Sprintf("GITBASEPG_DB_CONNECTION=root@tcp(%s)/none?maxAllowedPacket=4194304", gitbase.Name),
				fmt.Sprintf("GITBASEPG_BBLFSH_SERVER_URL=%s:%d", bblfshd.Name, components.BblfshParsePort),
				fmt.Sprintf("GITBASEPG_PORT=%d", components.GitbaseWebPort),
				fmt.Sprintf("GITBASEPG_SELECT_LIMIT=%d", gitbaseWebSelectLimit),
			},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbaseWeb.Name)
	}
}
