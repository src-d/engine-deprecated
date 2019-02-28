package main

import (
	"net"
	"strings"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd-server/engine"

	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

var version = "undefined"

func main() {
	var options struct {
		Addr    string `long:"address" short:"a" default:"0.0.0.0:4242"`
		Workdir string `long:"workdir" short:"w" default:""`
		Data    string `long:"data" short:"d" default:""`
		Config  string `long:"config" short:"c" default:""`
	}

	_, err := flags.Parse(&options)
	if err != nil {
		logrus.Fatal(err)
	}

	workdir := strings.TrimSpace(options.Workdir)
	if workdir == "" {
		logrus.Fatal("No work directory provided!")
	}

	datadir := strings.TrimSpace(options.Data)
	if datadir == "" {
		logrus.Fatal("No data directory provided!")
	}

	var config api.Config
	if options.Config != "" {
		err = yaml.Unmarshal([]byte(options.Config), &config)
		if err != nil {
			logrus.Fatalf("Error reading --config option: %s", err)
		}
	}
	config.SetDefaults()

	l, err := net.Listen("tcp", options.Addr)
	if err != nil {
		logrus.Fatal(err)
	}

	srv := grpc.NewServer()
	api.RegisterEngineServer(srv, engine.NewServer(version, workdir, datadir, config))

	logrus.Infof("listening on %s", options.Addr)
	if err := srv.Serve(l); err != nil {
		logrus.Fatal(err)
	}
}
