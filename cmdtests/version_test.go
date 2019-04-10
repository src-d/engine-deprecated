// +build integration

package cmdtests_test

import (
	"regexp"
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/suite"
)

type VersionTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestVersionTestSuite(t *testing.T) {
	s := VersionTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *VersionTestSuite) TestWithoutDaemon() {
	require := s.Require()

	r := s.RunCommand("version")
	require.NoError(r.Error, r.Combined())

	expected := regexp.MustCompile(
		`^srcd cli version: \S+
docker version: \S+
srcd daemon version: not running
$`)

	s.Regexp(expected, r.Stdout())
}

func (s *VersionTestSuite) TestWithDaemon() {
	require := s.Require()

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("version")
	require.NoError(r.Error, r.Combined())

	expected := regexp.MustCompile(
		`^srcd cli version: \S+
docker version: \S+
srcd daemon version: \S+
$`)

	s.Regexp(expected, r.Stdout())
}
