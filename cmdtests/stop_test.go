// +build integration

package cmdtests_test

import (
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/docker"

	"github.com/stretchr/testify/suite"
)

type StopTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestStopTestSuite(t *testing.T) {
	s := StopTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *StopTestSuite) TestInitStop() {
	require := s.Require()

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "SELECT 1")
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("stop")
	require.NoError(r.Error, r.Combined())

	s.AllStopped()
}

func (s *StopTestSuite) TestStopTwice() {
	require := s.Require()

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("stop")
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("stop")
	require.NoError(r.Error, r.Combined())
}

func (s *StopTestSuite) TestMissingContainers() {
	require := s.Require()

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	// start gitbase and bblfsh
	r = s.RunCommand("sql", "SELECT 1")
	require.NoError(r.Error, r.Combined())

	// kill the daemon container
	err := docker.RemoveContainer("srcd-cli-daemon")
	require.NoError(err)

	// run stop, the other containers should be stopped
	r = s.RunCommand("stop")
	require.NoError(r.Error, r.Combined())

	s.AllStopped()
}
