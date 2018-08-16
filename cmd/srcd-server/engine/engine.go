package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	bblfsh "gopkg.in/bblfsh/client-go.v2"
	enry "gopkg.in/src-d/enry.v1"

	drivers "github.com/bblfsh/bblfshd/daemon/protocol"
	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

var _ api.EngineServer = new(Server)

type Server struct {
	version string
}

func NewServer(version string) *Server {
	return &Server{
		version: version,
	}
}

func (s *Server) Version(ctx context.Context, req *api.VersionRequest) (*api.VersionResponse, error) {
	return &api.VersionResponse{Version: s.version}, nil
}

const (
	bblfshdName       = "srcd-cli-bblfshd"
	bblfshParsePort   = 9432
	bblfshControlPort = 9433
)

func (s *Server) Parse(ctx context.Context, req *api.ParseRequest) (*api.ParseResponse, error) {
	logrus.Infof("got parse request")
	lang := req.Lang
	if lang == "" {
		lang = enry.GetLanguage(req.Name, req.Content)
	}
	lang = strings.ToLower(lang)
	if req.Kind == api.ParseRequest_LANG {
		return &api.ParseResponse{Lang: lang}, nil
	}

	// TODO(campoy): this should be a bit more flexible, might need to a table somewhere.

	// check whether bblfshd is running or not
	_, err := docker.InfoOrStart(bblfshdName, createBbblfshd)
	if err != nil {
		return nil, err
	}

	// TODO(campoy): add security
	{
		addr := fmt.Sprintf("%s:%d", bblfshdName, bblfshControlPort)
		logrus.Infof("connecting to bblfsh management on %s", addr)
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrap(err, "could not connect to bblfsh drivers")
		}
		driverClient := drivers.NewProtocolServiceClient(conn)
		res, err := driverClient.InstallDriver(ctx, &drivers.InstallDriverRequest{
			Language: lang,
			// TODO(campoy): latest might not always be what we want, is it?
			ImageReference: fmt.Sprintf("docker://bblfsh/%s-driver:latest", lang),
		})

		j, _ := json.MarshalIndent(res, "", "  ")
		logrus.Infof("driver response: %s", j)

		if err != nil {
			return nil, errors.Wrap(err, "could not install driver")
		}

		// TODO(campoy): file an issue regarding this error, it should be in err above.
		if len(res.Errors) == 1 {
			if !strings.HasPrefix(res.Errors[0], "driver already installed") {
				return nil, fmt.Errorf(res.Errors[0])
			}
		} else if len(res.Errors) > 1 {
			return nil, fmt.Errorf("multiple errors: %s", strings.Join(res.Errors, "; "))
		}
	}

	addr := fmt.Sprintf("%s:%d", bblfshdName, bblfshParsePort)
	logrus.Infof("connecting to bblfsh parsing on %s", addr)
	client, err := bblfsh.NewClient(addr)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh")
	}

	res, err := client.NewParseRequest().
		Language(lang).
		Content(string(req.Content)).
		Filename(req.Name).
		DoWithContext(ctx)
	j, _ := json.MarshalIndent(res, "", "  ")
	logrus.Infof("parser response: %s", j)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse")
	}

	j, _ = json.MarshalIndent(res.UAST, "", "\t")
	logrus.Infof("uast: %s", j)

	uast, err := res.UAST.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal uast")
	}

	return &api.ParseResponse{Lang: lang, Uast: uast}, nil
}

func createBbblfshd() error {
	logrus.Infof("starting bblfshd daemon")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &container.Config{
		Image: "bblfsh/bblfshd",
		Cmd:   []string{"-ctl-address=0.0.0.0:9433", "-ctl-network=tcp"},
	}
	// TODO: add volume to store drivers
	host := &container.HostConfig{Privileged: true}

	return docker.Start(ctx, config, host, bblfshdName)
}
