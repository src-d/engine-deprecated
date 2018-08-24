package engine

import (
	"context"
	"fmt"
	"strings"

	drivers "github.com/bblfsh/bblfshd/daemon/protocol"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine-cli/api"
	"google.golang.org/grpc"
)

var ErrDriverAlreadyInstalled = errors.New("driver already installed")

func (s *Server) bblfshDriverClient() (drivers.ProtocolServiceClient, error) {
	if err := s.startComponent(bblfshd.Name); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", bblfshd.Name, bblfshControlPort)
	logrus.Infof("connecting to bblfsh management on %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh drivers")
	}

	return drivers.NewProtocolServiceClient(conn), nil
}

func (s *Server) ListDrivers(ctx context.Context, req *api.ListDriversRequest) (*api.ListDriversResponse, error) {
	client, err := s.bblfshDriverClient()
	if err != nil {
		return nil, err
	}

	res, err := client.DriverStates(ctx, &drivers.DriverStatesRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list drivers from bblfsh")
	}

	var list api.ListDriversResponse
	for _, state := range res.State {
		list.Drivers = append(list.Drivers, &api.ListDriversResponse_DriverInfo{
			Lang:    state.Language,
			Version: state.Version,
		})
	}

	return &list, nil
}

func (s *Server) InstallDriver(
	ctx context.Context,
	r *api.VersionedDriver,
) (*api.InstallDriverResponse, error) {
	err := s.installDriver(ctx, r.Language, r.Version, false)
	return new(api.InstallDriverResponse), err
}

func (s *Server) UpdateDriver(
	ctx context.Context,
	r *api.VersionedDriver,
) (*api.UpdateDriverResponse, error) {
	err := s.installDriver(ctx, r.Language, r.Version, true)
	return new(api.UpdateDriverResponse), err
}

func (s *Server) RemoveDriver(
	ctx context.Context,
	r *api.RemoveDriverRequest,
) (*api.RemoveDriverResponse, error) {
	client, err := s.bblfshDriverClient()
	if err != nil {
		return nil, err
	}

	_, err = client.RemoveDriver(ctx, &drivers.RemoveDriverRequest{Language: r.Language})
	return new(api.RemoveDriverResponse), err
}

func (s *Server) installDriver(ctx context.Context, lang, version string, update bool) error {
	client, err := s.bblfshDriverClient()
	if err != nil {
		return err
	}

	if update {
		_, err := client.RemoveDriver(ctx, &drivers.RemoveDriverRequest{Language: lang})
		if err != nil {
			return err
		}
	}

	resp, err := client.InstallDriver(ctx, &drivers.InstallDriverRequest{
		Language:       lang,
		ImageReference: driverImage(lang, version),
		Update:         update,
	})
	if err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		// TODO(campoy): file an issue regarding this error, it should be in err above.
		if !strings.HasPrefix(resp.Errors[0], "driver already installed") {
			return ErrDriverAlreadyInstalled
		}
		return fmt.Errorf("can't install %s driver: %s", lang, strings.Join(resp.Errors, "; "))
	}

	return nil
}

func driverImage(lang, version string) string {
	return fmt.Sprintf("docker://bblfsh/%s-driver:%s", lang, version)
}
