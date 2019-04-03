// +build integration

package cmdtests_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/suite"
)

type VersionTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestVersionTestSuite(t *testing.T) {
	s := VersionTestSuite{}
	suite.Run(t, &s)
}

func (s *VersionTestSuite) TestWithoutDaemon() {
	require := s.Require()

	buf, err := s.RunCommand(context.TODO(), "version")
	require.NoError(err)

	expected := regexp.MustCompile(
		`^srcd cli version: \S+
docker version: \S+
srcd daemon version: not running
$`)

	s.Regexp(expected, buf.String())
}

func (s *VersionTestSuite) TestWithDaemon() {
	require := s.Require()

	_, err := s.RunInit(context.TODO(), s.TestDir)
	require.NoError(err)

	buf, err := s.RunCommand(context.TODO(), "version")
	require.NoError(err)

	expected := regexp.MustCompile(
		`^srcd cli version: \S+
docker version: \S+
srcd daemon version: \S+
$`)

	s.Regexp(expected, buf.String())
}
