package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

const (
	gitbaseWebPrivatePort = 8080
	bblfshWebPrivatePort  = 80
)

var (
	gitbaseWeb = components.GitbaseWeb
	bblfshWeb  = components.BblfshWeb
)

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
				fmt.Sprintf("GITBASEPG_PORT=%d", gitbaseWebPrivatePort),
			},
		}
		host := &container.HostConfig{}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, gitbaseWeb.Name)
	}
}
