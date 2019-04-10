// +build integration

package cmdtests_test

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// This test suite by default does not test the `--with-images` flag.
// If the tests are run with `make test-integration` the daemon image will be
// the one build locally (e.g. srcd/cli-daemon:dev-b72f1fe), and deleting
// this image would make all the other tests fail.
// Running the tests with `TEST_PRUNE_WITH_IMAGE=true make test-integration`
// will run the test with `--with-images` flag as last in order to avoid the
// aforementioned problem.

type PruneTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func (s *PruneTestSuite) testRunningContainers(require *require.Assertions, withImages bool) {
	s.T().Helper()

	// Get the list of volumes and networks before calling init
	prevVols, err := docker.ListVolumes(context.Background())
	require.NoError(err)

	prevNets, err := listNetworks()
	require.NoError(err)

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "SELECT 1")
	require.NoError(r.Error, r.Combined())

	if withImages {
		r = s.RunCommand("prune", "--with-images")
	} else {
		r = s.RunCommand("prune")
	}

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

func (s *PruneTestSuite) TestRunningContainers() {
	s.testRunningContainers(s.Require(), false)
}

func (s *PruneTestSuite) TestStoppedContainers() {
	require := s.Require()

	r := s.RunCommand("prune")
	require.NoError(r.Error, r.Combined())
}

func (s *PruneTestSuite) TestRunningContainersWithImages() {
	if os.Getenv("TEST_PRUNE_WITH_IMAGE") != "true" {
		s.T().Skip("Use env var TEST_PRUNE_WITH_IMAGE=true to test srcd prune with --with-images flag")
	}

	s.testRunningContainers(s.Require(), true)
	s.requireNoImages()
}

func (s *PruneTestSuite) requireNoImages() {
	s.T().Helper()
	require := s.Require()

	images, _ := docker.ListImages(context.Background())
	imagesNames := make(map[string]components.Component)
	for _, c := range []components.Component{
		components.Daemon,
		components.Gitbase,
		components.GitbaseWeb,
		components.MysqlCli,
		components.Bblfshd,
		components.BblfshWeb,
	} {
		imagesNames[c.Image] = c
	}

	for _, i := range images {
		for _, rt := range i.RepoTags {
			img, _ := docker.SplitImageID(rt)
			comp, ok := imagesNames[img]
			require.Falsef(ok, "Image with repo tag '%s' shouldn't be present for component '%s'", rt, comp.Name)
		}
	}
}

func TestPruneTestSuite(t *testing.T) {
	suite.Run(t, &PruneTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()})
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
	c, err := docker.GetClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not create docker client")
	}

	return c.NetworkList(context.Background(), types.NetworkListOptions{})
}
