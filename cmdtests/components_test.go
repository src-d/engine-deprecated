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
	s := ComponentsTestSuite{}
	suite.Run(t, &s)
}

func (s *ComponentsTestSuite) TestListStopped() {
	require := s.Require()

	out, err := s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	expected := regexp.MustCompile(
		`^IMAGE +INSTALLED +RUNNING +PORT +CONTAINER NAME
bblfsh/bblfshd:\S+ +(yes|no) +no +(\d+)? +srcd-cli-bblfshd
bblfsh/web:\S+ +(yes|no) +no +(\d+)? +srcd-cli-bblfsh-web
mysql:\S+ +(yes|no) +no +(\d+)? +srcd-cli-mysql-cli
srcd/cli-daemon:\S+ +(yes|no) +no +(\d+)? +srcd-cli-daemon
srcd/gitbase-web:\S+ +(yes|no) +no +(\d+)? +srcd-cli-gitbase-web
srcd/gitbase:\S+ +(yes|no) +no +(\d+)? +srcd-cli-gitbase
$`)

	s.Regexp(expected, out.String())
}

func (s *ComponentsTestSuite) TestListInit() {
	require := s.Require()

	out, err := s.RunInit(context.TODO(), s.TestDir)
	require.NoError(err, out.String())

	out, err = s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	expected := regexp.MustCompile(`srcd/cli-daemon:\S+ +yes +yes +4252 +srcd-cli-daemon`)
	s.Regexp(expected, out.String())
}

func (s *ComponentsTestSuite) TestInstall() {
	require := s.Require()

	out, err := s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	// Get the exact image:version of gitbase
	exp := regexp.MustCompile(`(srcd/gitbase:\S+) +(yes|no)`)
	matches := exp.FindStringSubmatch(out.String())

	require.NotNil(matches)
	require.Len(matches, 3)

	imgVersion := matches[1]
	installed := matches[2]

	// If installed, remove it
	if installed == "yes" {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		err = docker.RemoveImage(ctx, imgVersion)
		require.NoError(err)
	}

	// Check it's not installed
	out, err = s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	expected := regexp.MustCompile(`srcd/gitbase:\S+ +no +no +srcd-cli-gitbase`)
	require.Regexp(expected, out.String())

	// Install
	out, err = s.RunCommand(context.TODO(), "components", "install", "srcd/gitbase")
	require.NoError(err, out.String())

	// Check it's installed
	out, err = s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	expected = regexp.MustCompile(`srcd/gitbase:\S+ +yes +no +srcd-cli-gitbase`)
	require.Regexp(expected, out.String())

	// Call install again, should be an exit 0
	out, err = s.RunCommand(context.TODO(), "components", "install", "srcd/gitbase")
	require.NoError(err, out.String())
}

func (s *ComponentsTestSuite) TestInstallAlias() {
	require := s.Require()

	// Install with image name
	out, err := s.RunCommand(context.TODO(), "components", "install", "srcd/gitbase")
	require.NoError(err, out.String())

	// Install with container name
	out, err = s.RunCommand(context.TODO(), "components", "install", "srcd-cli-gitbase")
	require.NoError(err, out.String())
}

func (s *ComponentsTestSuite) TestInstallUnknown() {
	require := s.Require()

	// Call install with a srcd image not managed by engine
	out, err := s.RunCommand(context.TODO(), "components", "install", "srcd/lookout")
	require.Error(err)
	require.Contains(out.String(), "srcd/lookout is not valid. Component must be one of")
}

func (s *ComponentsTestSuite) TestInstallVersion() {
	require := s.Require()

	out, err := s.RunCommand(context.TODO(), "components", "list")
	require.NoError(err, out.String())

	// Get the exact image:version of gitbase
	exp := regexp.MustCompile(`(srcd/gitbase:\S+)`)
	matches := exp.FindStringSubmatch(out.String())

	require.NotNil(matches)
	require.Len(matches, 2)

	imgVersion := matches[1]

	// Call install with image:version
	out, err = s.RunCommand(context.TODO(), "components", "install", imgVersion)
	require.Error(err)
	require.Contains(out.String(), imgVersion+" is not valid. Component must be one of")
}
