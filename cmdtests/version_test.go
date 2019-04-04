// +build integration

package cmdtests_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/suite"
)

type VersionTestSuite struct {
	cmdtests.IntegrationSuite
	testDir string
}

func TestVersionTestSuite(t *testing.T) {
	s := VersionTestSuite{}
	suite.Run(t, &s)
}

func (s *VersionTestSuite) SetupTest() {
	var err error
	s.testDir, err = ioutil.TempDir("", "version-test")
	if err != nil {
		log.Fatal(err)
	}
}

func (s *VersionTestSuite) TearDownTest() {
	s.RunStop(context.Background())
	os.RemoveAll(s.testDir)
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

	_, err := s.RunInit(context.TODO(), s.testDir)
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
