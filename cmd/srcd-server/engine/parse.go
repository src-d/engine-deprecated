package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
	bblfsh "gopkg.in/bblfsh/client-go.v2"
	"gopkg.in/bblfsh/client-go.v2/tools"
	"gopkg.in/bblfsh/sdk.v1/uast"
	enry "gopkg.in/src-d/enry.v1"
)

const (
	bblfshMountPath   = "/var/lib/bblfshd"
	bblfshParsePort   = 9432
	bblfshControlPort = 9433
)

var bblfshd = components.Bblfshd

type logf func(format string, args ...interface{})

func (s *Server) ParseWithLogs(req *api.ParseRequest, stream api.Engine_ParseWithLogsServer) error {
	log := func(format string, args ...interface{}) {
		logrus.Infof(format, args...)
		err := stream.Send(&api.ParseResponse{
			Kind: api.ParseResponse_LOG,
			Log:  fmt.Sprintf(format, args...),
		})
		if err != nil {
			logrus.Errorf("could not stream log: %v", err)
		}
	}

	res, err := s.parse(stream.Context(), req, log)
	if err != nil {
		return err
	}
	return stream.Send(res)
}

func (s *Server) Parse(ctx context.Context, req *api.ParseRequest) (*api.ParseResponse, error) {
	return s.parse(ctx, req, logrus.Infof)
}

func (s *Server) parse(ctx context.Context, req *api.ParseRequest, log logf) (*api.ParseResponse, error) {
	log("got parse request")
	lang := req.Lang
	if lang == "" {
		lang = enry.GetLanguage(req.Name, req.Content)
	}
	lang = strings.ToLower(lang)
	if req.Kind == api.ParseRequest_LANG {
		return &api.ParseResponse{Lang: lang}, nil
	}

	// TODO(campoy): this should be a bit more flexible, might need to a table somewhere.

	if err := s.startComponent(bblfshd.Name); err != nil {
		return nil, err
	}

	dclient, err := s.bblfshDriverClient()
	if err != nil {
		return nil, err
	}

	err = s.installDriver(ctx, dclient, lang, "latest", false)
	if err == ErrDriverAlreadyInstalled {
		log("driver was already installed")
	} else if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", bblfshd.Name, bblfshParsePort)
	log("connecting to bblfsh parsing on %s", addr)
	client, err := bblfsh.NewClient(addr)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh")
	}

	res, err := client.NewParseRequest().
		Language(lang).
		Content(string(req.Content)).
		Filename(req.Name).
		DoWithContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse")
	}

	var nodes = []*uast.Node{res.UAST}
	if req.Query != "" {
		filtered, err := tools.Filter(res.UAST, req.Query)
		if err != nil {
			return nil, errors.Wrapf(err, "could not apply query %s", req.Query)
		}
		nodes = filtered
	}

	resp := &api.ParseResponse{Kind: api.ParseResponse_FINAL, Lang: lang}
	for _, node := range nodes {
		uast, err := node.Marshal()
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal uast")
		}
		resp.Uast = append(resp.Uast, uast)
	}
	return resp, nil
}

func createBbblfshd(setupFunc func() error, opts ...docker.ConfigOption) docker.StartFunc {
	return func() error {
		if err := docker.EnsureInstalled(bblfshd.Image, ""); err != nil {
			return err
		}

		if err := docker.CreateVolume(context.Background(), components.BblfshVolume); err != nil {
			return err
		}

		logrus.Infof("starting bblfshd daemon")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		config := &container.Config{
			Image: bblfshd.Image,
			Cmd:   []string{"-ctl-address=0.0.0.0:9433", "-ctl-network=tcp"},
		}

		host := &container.HostConfig{Privileged: true}
		docker.ApplyOptions(config, host, opts...)

		if err := docker.Start(ctx, config, host, bblfshd.Name); err != nil {
			return err
		}

		return setupFunc()
	}
}
