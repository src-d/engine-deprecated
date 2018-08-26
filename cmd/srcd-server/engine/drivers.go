package engine

import (
	"context"
	"encoding/json"
	"fmt"

	drivers "github.com/bblfsh/bblfshd/daemon/protocol"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/src-d/engine-cli/api"
	"google.golang.org/grpc"
)

func (s *Server) ListDrivers(ctx context.Context, req *api.ListDriversRequest) (*api.ListDriversResponse, error) {
	if err := Run(Component{Name: bblfshd.Name, Start: createBbblfshd}); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", bblfshd.Name, bblfshControlPort)
	logrus.Infof("connecting to bblfsh management on %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to bblfsh drivers")
	}
	client := drivers.NewProtocolServiceClient(conn)
	res, err := client.DriverStates(ctx, &drivers.DriverStatesRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list drivers from bblfsh")
	}

	j, _ := json.MarshalIndent(res, "", "  ")
	logrus.Infof("response: %s", j)

	var list api.ListDriversResponse
	for _, state := range res.State {
		list.Drivers = append(list.Drivers, &api.ListDriversResponse_DriverInfo{
			Lang:    state.Language,
			Version: state.Version,
		})
	}
	return &list, nil
}
