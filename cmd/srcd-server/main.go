package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd-server/engine"

	flags "github.com/jessevdk/go-flags"
	grpc "google.golang.org/grpc"
	"gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

var version = "undefined"

func main() {
	var options struct {
		Addr    string `long:"address" short:"a" default:"0.0.0.0:4242"`
		Workdir string `long:"workdir" short:"w" default:""`
		HostOS  string `long:"host-os" default:""`
		Config  string `long:"config" short:"c" default:""`
	}

	_, err := flags.Parse(&options)
	if err != nil {
		log.Errorf(err, "")
		os.Exit(1)
	}

	workdir := strings.TrimSpace(options.Workdir)
	if workdir == "" {
		log.Errorf(fmt.Errorf("No work directory provided!"), "")
		os.Exit(1)
	}

	var config api.Config
	if options.Config != "" {
		err = yaml.Unmarshal([]byte(options.Config), &config)
		if err != nil {
			log.Errorf(err, "Error reading --config option: %s")
			os.Exit(1)
		}
	}
	config.SetDefaults()

	l, err := net.Listen("tcp", options.Addr)
	if err != nil {
		log.Errorf(err, "")
		os.Exit(1)
	}

	srv := grpc.NewServer()
	api.RegisterEngineServer(srv, engine.NewServer(version, workdir, options.HostOS, config))

	log.Infof("listening on %s", options.Addr)
	if err := srv.Serve(l); err != nil {
		log.Errorf(err, "")
		os.Exit(1)
	}
}
