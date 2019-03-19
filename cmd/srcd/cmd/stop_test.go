// +build integration

package cmd

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	cmdtest "github.com/src-d/engine/cmd/test-utils"
	"github.com/src-d/engine/docker"

	"github.com/stretchr/testify/suite"
)

type StopTestSuite struct {
	cmdtest.IntegrationSuite
	testDir string
}

func TestStopTestSuite(t *testing.T) {
	s := StopTestSuite{}
	suite.Run(t, &s)
}

func (s *StopTestSuite) SetupTest() {
	var err error
	s.testDir, err = ioutil.TempDir("", "stop-test")
	if err != nil {
		log.Fatal(err)
	}
}

func (s *StopTestSuite) TearDownTest() {
	s.RunStop(context.Background())
	os.RemoveAll(s.testDir)
}

func (s *StopTestSuite) TestInitStop() {
	require := s.Require()

	_, err := s.RunInit(context.TODO(), s.testDir)
	require.NoError(err)

	_, err = s.RunSQL(context.TODO(), "SELECT 1")
	require.NoError(err)

	_, err = s.RunStop(context.TODO())
	require.NoError(err)

	s.AllStopped()
}

func (s *StopTestSuite) TestStopTwice() {
	require := s.Require()

	_, err := s.RunInit(context.TODO(), s.testDir)
	require.NoError(err)

	_, err = s.RunStop(context.TODO())
	require.NoError(err)

	_, err = s.RunStop(context.TODO())
	require.NoError(err)
}

func (s *StopTestSuite) TestMissingContainers() {
	require := s.Require()

	_, err := s.RunInit(context.TODO(), s.testDir)
	require.NoError(err)

	// start gitbase and bblfsh
	_, err = s.RunSQL(context.TODO(), "SELECT 1")
	require.NoError(err)

	// kill the daemon container
	err = docker.RemoveContainer("srcd-cli-daemon")
	require.NoError(err)

	// run stop, the other containers should be stopped
	_, err = s.RunStop(context.TODO())
	require.NoError(err)

	s.AllStopped()
}
