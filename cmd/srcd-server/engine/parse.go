package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
	bblfsh "gopkg.in/bblfsh/client-go.v3"
	"gopkg.in/bblfsh/client-go.v3/tools"
	"gopkg.in/bblfsh/sdk.v2/uast/nodes"
	enry "gopkg.in/src-d/enry.v1"
)

const (
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

	if err := s.startComponent(ctx, bblfshd.Name); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", bblfshd.Name, bblfshParsePort)
	log("connecting to bblfsh parsing on %s", addr)
	client, err := bblfsh.NewClient(addr)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh")
	}

	mode := bblfsh.Semantic
	switch req.Mode {
	case api.ParseRequest_ANNOTATED:
		mode = bblfsh.Annotated
	case api.ParseRequest_NATIVE:
		mode = bblfsh.Native
	}

	res, _, err := client.NewParseRequest().
		Language(lang).
		Content(string(req.Content)).
		Filename(req.Name).
		Context(ctx).
		Mode(mode).
		UAST()
	if err != nil {
		return nil, errors.Wrap(err, "could not parse")
	}

	var ns = []nodes.Node{res}
	if req.Query != "" {
		var filtered nodes.Array
		iter, err := tools.Filter(res, req.Query)
		if err != nil {
			return nil, errors.Wrapf(err, "could not apply query %s", req.Query)
		}
		for iter.Next() {
			filtered = append(filtered, iter.Node().(nodes.Node))
		}
		ns = filtered
	}

	resp := &api.ParseResponse{Kind: api.ParseResponse_FINAL, Lang: lang}
	for _, node := range ns {
		uast, err := json.MarshalIndent(node, "", "  ")
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal uast")
		}
		resp.Uast = append(resp.Uast, uast)
	}
	return resp, nil
}

func createBbblfshd(opts ...docker.ConfigOption) docker.StartFunc {
	return func(ctx context.Context) error {
		if err := docker.EnsureInstalled(bblfshd.Image, bblfshd.Version); err != nil {
			return err
		}

		logrus.Infof("starting bblfshd daemon")

		ctx, cancel := context.WithTimeout(ctx, startComponentTimeout)
		defer cancel()

		config := &container.Config{
			Image: bblfshd.ImageWithVersion(),
			Cmd:   []string{"-ctl-address=0.0.0.0:9433", "-ctl-network=tcp"},
		}

		host := &container.HostConfig{Privileged: true}
		docker.ApplyOptions(config, host, opts...)

		return docker.Start(ctx, config, host, bblfshd.Name)
	}
}
