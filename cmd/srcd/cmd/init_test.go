// +build integration

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	cmdtest "github.com/src-d/engine/cmd/test-utils"
	"github.com/src-d/engine/components"
	"github.com/stretchr/testify/suite"
)

type InitTestSuite struct {
	cmdtest.IntegrationSuite
	timeout        time.Duration
	testDir        string
	validWorkDir   string
	invalidWorkDir string
}

func TestInitTestSuite(t *testing.T) {
	itt := InitTestSuite{timeout: 1 * time.Minute}
	suite.Run(t, &itt)
}

func (s *InitTestSuite) SetupTest() {
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", ".integration-testing"))
	if err != nil {
		log.Fatal(err)
	}

	s.testDir = dir
	s.validWorkDir = filepath.Join(s.testDir, "valid-workdir")
	s.invalidWorkDir = filepath.Join(s.testDir, "invalid-workdir")

	err = os.Mkdir(s.testDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(s.validWorkDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Create(s.invalidWorkDir)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *InitTestSuite) TearDownTest() {
	s.RunStop(context.Background())
	os.RemoveAll(s.testDir)
}

func (s *InitTestSuite) runInit(workdir string) (*bytes.Buffer, error) {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.RunInit(ctx, workdir)
}

func (s *InitTestSuite) runSQL() (*bytes.Buffer, error) {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.RunSQL(ctx, "select 1")
}

func (s *InitTestSuite) getLogMessages(buf *bytes.Buffer) []string {
	actualMsg := s.ParseLogMessages(buf)
	var filteredMsg []string
	for _, m := range actualMsg {
		if m == "unable to list the available daemon versions on Docker Hub: Short version cannot contain PreRelease/Build meta data" {
			continue
		}

		filteredMsg = append(filteredMsg, m)
	}

	return filteredMsg
}

func (s *InitTestSuite) TestWithoutWorkdir() {
	require := s.Require()

	buf, err := s.runInit("")
	require.NoError(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(2, len(actualMsg))

	workdir, _ := os.Getwd()
	expectedMsg := [2]string{
		fmt.Sprintf("starting daemon with working directory: %s", workdir),
		"daemon started",
	}

	for i, exp := range expectedMsg {
		require.Equal(exp, actualMsg[i])
	}
}

func (s *InitTestSuite) TestWithValidWorkdir() {
	require := s.Require()

	buf, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(2, len(actualMsg))

	expectedMsg := [2]string{
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for i, exp := range expectedMsg {
		require.Equal(exp, actualMsg[i])
	}
}

func (s *InitTestSuite) TestWithInvalidWorkdir() {
	require := s.Require()

	buf, err := s.runInit(s.invalidWorkDir)
	require.Error(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(1, len(actualMsg))

	expectedMsg := [1]string{
		fmt.Sprintf("path \\\"%s\\\" is not a valid working directory", s.invalidWorkDir),
	}

	for i, exp := range expectedMsg {
		require.Equal(exp, actualMsg[i])
	}
}

func (s *InitTestSuite) TestWithRunningDaemon() {
	require := s.Require()

	_, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	buf, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(3, len(actualMsg))

	expectedMsg := [3]string{
		fmt.Sprintf("removing container %s", components.Daemon.Name),
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for i, exp := range expectedMsg {
		require.Equal(exp, actualMsg[i])
	}
}

func (s *InitTestSuite) TestWithRunningOtherComponents() {
	require := s.Require()

	_, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	_, err = s.runSQL()
	require.NoError(err)

	buf, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(5, len(actualMsg))

	expectedMsg := [5]string{
		fmt.Sprintf("removing container %s", components.Bblfshd.Name),
		fmt.Sprintf("removing container %s", components.Daemon.Name),
		fmt.Sprintf("removing container %s", components.Gitbase.Name),
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for i, exp := range expectedMsg {
		require.Equal(exp, actualMsg[i])
	}
}
