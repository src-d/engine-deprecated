package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	bblfsh "gopkg.in/bblfsh/client-go.v2"
	"gopkg.in/bblfsh/client-go.v2/tools"
	"gopkg.in/bblfsh/sdk.v1/uast"
	enry "gopkg.in/src-d/enry.v1"

	drivers "github.com/bblfsh/bblfshd/daemon/protocol"
	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/docker"
)

var _ api.EngineServer = new(Server)

type Server struct {
	version string
	workdir string
}

func NewServer(version, workdir string) *Server {
	return &Server{
		version: version,
		workdir: workdir,
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

	if err := Run(Component{Name: bblfshdName, Start: createBbblfshd}); err != nil {
		return nil, err
	}

	// TODO(campoy): add security
	{
		addr := fmt.Sprintf("%s:%d", bblfshdName, bblfshControlPort)
		log("connecting to bblfsh management on %s", addr)
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrap(err, "could not connect to bblfsh drivers")
		}
		log("installing driver for %s", lang)
		driverClient := drivers.NewProtocolServiceClient(conn)
		res, err := driverClient.InstallDriver(ctx, &drivers.InstallDriverRequest{
			Language: lang,
			// TODO(campoy): latest might not always be what we want, is it?
			ImageReference: fmt.Sprintf("docker://bblfsh/%s-driver:latest", lang),
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not install driver")
		}

		// TODO(campoy): file an issue regarding this error, it should be in err above.
		if len(res.Errors) == 1 {
			if !strings.HasPrefix(res.Errors[0], "driver already installed") {
				return nil, fmt.Errorf(res.Errors[0])
			}
			log("driver was already installed")
		} else if len(res.Errors) > 1 {
			return nil, fmt.Errorf("multiple errors: %s", strings.Join(res.Errors, "; "))
		}
	}

	addr := fmt.Sprintf("%s:%d", bblfshdName, bblfshParsePort)
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

func (s *Server) withWorkdirMounted(at string) docker.ConfigOption {
	return docker.WithVolume(s.workdir, at)
}

func createBbblfshd() error {
	if err := docker.EnsureInstalled(gitbaseImage, ""); err != nil {
		return err
	}

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
