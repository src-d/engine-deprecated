// +build integration

package cmdtests_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/docker"

	"github.com/stretchr/testify/suite"
)

type ComponentsTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestComponentsTestSuite(t *testing.T) {
	s := ComponentsTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *ComponentsTestSuite) TestListStopped() {
	require := s.Require()

	r := s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	expected := regexp.MustCompile(
		`^IMAGE +INSTALLED +RUNNING +PORT +CONTAINER NAME
bblfsh/bblfshd:\S+ +(yes|no) +no +(\d+)? +srcd-cli-bblfshd
bblfsh/web:\S+ +(yes|no) +no +(\d+)? +srcd-cli-bblfsh-web
mysql:\S+ +(yes|no) +no +(\d+)? +srcd-cli-mysql-cli
srcd/cli-daemon:\S+ +(yes|no) +no +(\d+)? +srcd-cli-daemon
srcd/gitbase-web:\S+ +(yes|no) +no +(\d+)? +srcd-cli-gitbase-web
srcd/gitbase:\S+ +(yes|no) +no +(\d+)? +srcd-cli-gitbase
$`)

	s.Regexp(expected, r.Combined())
}

func (s *ComponentsTestSuite) TestListInit() {
	require := s.Require()

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	expected := regexp.MustCompile(`srcd/cli-daemon:\S+ +yes +yes +4252 +srcd-cli-daemon`)
	s.Regexp(expected, r.Stdout())
}

func (s *ComponentsTestSuite) TestInstall() {
	require := s.Require()

	r := s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	// Get the exact image:version of gitbase
	exp := regexp.MustCompile(`(srcd/gitbase:\S+) +(yes|no)`)
	matches := exp.FindStringSubmatch(r.Stdout())

	require.NotNil(matches)
	require.Len(matches, 3)

	imgVersion := matches[1]
	installed := matches[2]

	// If installed, remove it
	if installed == "yes" {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		require.NoError(docker.RemoveImage(ctx, imgVersion))
	}

	// Check it's not installed
	r = s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	expected := regexp.MustCompile(`srcd/gitbase:\S+ +no +no +srcd-cli-gitbase`)
	require.Regexp(expected, r.Stdout())

	// Install
	r = s.RunCommand("components", "install", "srcd/gitbase")
	require.NoError(r.Error, r.Combined())

	// Check it's installed
	r = s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	expected = regexp.MustCompile(`srcd/gitbase:\S+ +yes +no +srcd-cli-gitbase`)
	require.Regexp(expected, r.Stdout())

	// Call install again, should be an exit 0
	r = s.RunCommand("components", "install", "srcd/gitbase")
	require.NoError(r.Error, r.Combined())
}

func (s *ComponentsTestSuite) TestInstallAlias() {
	require := s.Require()

	// Install with image name
	r := s.RunCommand("components", "install", "srcd/gitbase")
	require.NoError(r.Error, r.Combined())

	// Install with container name
	r = s.RunCommand("components", "install", "srcd-cli-gitbase")
	require.NoError(r.Error, r.Combined())
}

func (s *ComponentsTestSuite) TestInstallUnknown() {
	require := s.Require()

	// Call install with a srcd image not managed by engine
	r := s.RunCommand("components", "install", "srcd/lookout")
	require.Error(r.Error)
	require.Contains(r.Stderr(), "srcd/lookout is not valid. Component must be one of")
}

func (s *ComponentsTestSuite) TestInstallVersion() {
	require := s.Require()

	r := s.RunCommand("components", "list")
	require.NoError(r.Error, r.Combined())

	// Get the exact image:version of gitbase
	exp := regexp.MustCompile(`(srcd/gitbase:\S+)`)
	matches := exp.FindStringSubmatch(r.Stdout())

	require.NotNil(matches)
	require.Len(matches, 2)

	imgVersion := matches[1]

	// Call install with image:version
	r = s.RunCommand("components", "install", imgVersion)
	require.Error(r.Error)
	require.Contains(r.Stderr(), imgVersion+" is not valid. Component must be one of")
}
