package engine

import (
	"context"
	"fmt"

	drivers "github.com/bblfsh/bblfshd/daemon/protocol"
	"github.com/pkg/errors"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/components"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-log.v1"
)

var ErrDriverAlreadyInstalled = errors.New("driver already installed")

func (s *Server) bblfshDriverClient(ctx context.Context) (drivers.ProtocolServiceClient, error) {
	if err := s.startComponent(ctx, bblfshd.Name); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", bblfshd.Name, components.BblfshControlPort)
	log.Infof("connecting to bblfsh management on %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh drivers")
	}

	return drivers.NewProtocolServiceClient(conn), nil
}

func (s *Server) ListDrivers(ctx context.Context, req *api.ListDriversRequest) (*api.ListDriversResponse, error) {
	client, err := s.bblfshDriverClient(ctx)
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
