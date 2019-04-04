// +build integration

package cmdtests_test

import (
	"context"
	"sort"
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/docker"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

// This test suite does not test the `--with-images` flag.
// If the tests are run with `make test-integration` the daemon image will be
// the one build locally (e.g. srcd/cli-daemon:dev-b72f1fe), and deleting
// this image would make all the other tests fail.

type PruneTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestPruneTestSuite(t *testing.T) {
	s := PruneTestSuite{}
	suite.Run(t, &s)
}

func (s *PruneTestSuite) TestRunningContainers() {
	require := s.Require()

	// Get the list of volumes and networks before calling init
	prevVols, err := docker.ListVolumes(context.Background())
	require.NoError(err)

	prevNets, err := listNetworks()
	require.NoError(err)

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "SELECT 1")
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("prune")
	require.NoError(r.Error, r.Combined())

	// Test containers were deleted
	s.AllStopped()

	// Test volumes with name srcd-cli-* were deleted.
	// The logic used in prune to delete named volumes is looking for the
	// srcd-cli- prefix in the name, so that's what we check here.
	vols, err := docker.ListVolumes(context.Background())
	require.NoError(err)

	for _, vol := range vols {
		require.NotContainsf(vol.Name, "srcd-cli-", "Volume %s was not deleted", vol.Name)
	}

	// Test anonymous volumes were deleted
	require.Equal(volNames(prevVols), volNames(vols))

	// Test srcd-cli-network network was deleted.
	nets, err := listNetworks()
	require.NoError(err)

	for _, net := range nets {
		require.NotEqualf(net.Name, docker.NetworkName, "Network %s was not deleted", net.Name)
	}

	// Test any other user-defined networks were deleted
	require.Equal(netNames(prevNets), netNames(nets))
}

func volNames(volumes []*docker.Volume) []string {
	var names []string
	for _, vol := range volumes {
		names = append(names, vol.Name)
	}

	sort.Strings(names)
	return names
}

func netNames(nets []types.NetworkResource) []string {
	var names []string
	for _, net := range nets {
		names = append(names, net.Name)
	}

	sort.Strings(names)
	return names
}

func listNetworks() ([]types.NetworkResource, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	return c.NetworkList(context.Background(), types.NetworkListOptions{})
}

func (s *PruneTestSuite) TestStoppedContainers() {
	require := s.Require()

	r := s.RunCommand("prune")
	require.NoError(r.Error, r.Combined())
}
