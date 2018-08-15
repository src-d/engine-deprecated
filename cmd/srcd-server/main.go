package main

import (
	"net"

	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/cmd/srcd-server/engine"
	grpc "google.golang.org/grpc"
)

var version = "undefined"

func main() {
	var options struct {
		Addr string `long:"address" short:"a" default:"0.0.0.0:4242"`
	}
	_, err := flags.Parse(&options)
	if err != nil {
		logrus.Fatal(err)
	}

	l, err := net.Listen("tcp", options.Addr)
	if err != nil {
		logrus.Fatal(err)
	}
	srv := grpc.NewServer()
	api.RegisterEngineServer(srv, engine.NewServer(version))

	logrus.Infof("listening on %s", options.Addr)
	if err := srv.Serve(l); err != nil {
		logrus.Fatal(err)
	}
}
