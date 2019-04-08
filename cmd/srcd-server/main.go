package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd-server/engine"

	"github.com/pkg/errors"
	grpc "google.golang.org/grpc"
	"gopkg.in/src-d/go-cli.v0"
	"gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

// These variables get replaced during the build
var (
	version = "dev"
	build   = "dev"
)

func main() {
	cmd := cli.New("srcd-server", version, build, "The Code as Data solution by source{d}")
	cmd.AddCommand(&serveCmd{})

	cmd.RunMain()
}

type serveCmd struct {
	cli.Command `name:"serve" short-description:"Start the server" long-description:"Start the server"`

	Addr    string `long:"address" short:"a" default:"0.0.0.0:4242"`
	Workdir string `long:"workdir" short:"w" default:""`
	HostOS  string `long:"host-os" default:""`
	Config  string `long:"config" short:"c" default:""`
}

func (c *serveCmd) Execute(args []string) error {
	workdir := strings.TrimSpace(c.Workdir)
	if workdir == "" {
		return fmt.Errorf("No work directory provided!")
	}

	var config api.Config
	if c.Config != "" {
		err := yaml.Unmarshal([]byte(c.Config), &config)
		if err != nil {
			return errors.Wrapf(err, "Error reading --config option")
		}
	}
	config.SetDefaults()

	l, err := net.Listen("tcp", c.Addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	api.RegisterEngineServer(srv, engine.NewServer(version, workdir, c.HostOS, config))

	log.Infof("listening on %s", c.Addr)
	return srv.Serve(l)
}
