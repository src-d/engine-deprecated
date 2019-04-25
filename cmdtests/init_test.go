// +build integration

package cmdtests_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/components"
	"github.com/stretchr/testify/suite"
)

type InitTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
	timeout        time.Duration
	validWorkDir   string
	invalidWorkDir string
}

func TestInitTestSuite(t *testing.T) {
	itt := InitTestSuite{
		IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite(),
		timeout:                1 * time.Minute,
	}
	suite.Run(t, &itt)
}

func (s *InitTestSuite) SetupTest() {
	s.IntegrationTmpDirSuite.SetupTest()

	s.validWorkDir = filepath.Join(s.TestDir, "valid-workdir")
	s.invalidWorkDir = filepath.Join(s.TestDir, "invalid-workdir")

	err := os.MkdirAll(s.validWorkDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Create(s.invalidWorkDir)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *InitTestSuite) getLogMessages(output string) []string {
	actualMsg := s.ParseLogMessages(output)
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

	r := s.RunInit("")
	require.NoError(r.Error, r.Combined())

	actualMsg := s.getLogMessages(r.Combined())

	workdir, _ := os.Getwd()
	expectedMsg := [2]string{
		fmt.Sprintf("starting daemon with working directory: %s", workdir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) TestWithValidWorkdir() {
	require := s.Require()

	r := s.RunInit(s.validWorkDir)
	require.NoError(r.Error, r.Combined())

	actualMsg := s.getLogMessages(r.Combined())

	expectedMsg := [2]string{
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) TestWithInvalidWorkdir() {
	require := s.Require()

	r := s.RunInit(s.invalidWorkDir)
	require.Error(r.Error)

	require.Equal(
		fmt.Sprintf("path '%s' is not a valid working directory\n", s.invalidWorkDir),
		r.Stderr(),
	)
}

func (s *InitTestSuite) TestWithRunningDaemon() {
	require := s.Require()

	r := s.RunInit(s.validWorkDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunInit(s.validWorkDir)
	require.NoError(r.Error, r.Combined())

	actualMsg := s.getLogMessages(r.Combined())

	expectedMsg := [3]string{
		fmt.Sprintf("removing container %s", components.Daemon.Name),
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) TestWithRunningOtherComponents() {
	require := s.Require()

	r := s.RunInit(s.validWorkDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select 1")
	require.NoError(r.Error, r.Combined())

	r = s.RunInit(s.validWorkDir)
	require.NoError(r.Error, r.Combined())

	actualMsg := s.getLogMessages(r.Combined())

	expectedMsg := [5]string{
		fmt.Sprintf("removing container %s", components.Bblfshd.Name),
		fmt.Sprintf("removing container %s", components.Daemon.Name),
		fmt.Sprintf("removing container %s", components.Gitbase.Name),
		fmt.Sprintf("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) initGitRepo(path string) {
	s.T().Helper()

	err := os.MkdirAll(path, os.ModePerm)
	s.Require().NoError(err)

	cmd := exec.Command("git", "init", path)
	err = cmd.Run()
	s.Require().NoError(err)
}

func (s *InitTestSuite) TestChangeWorkdir() {
	require := s.Require()

	// Create 2 workdirs, each with a repo
	workdirA := filepath.Join(s.TestDir, "workdir_a")
	workdirB := filepath.Join(s.TestDir, "workdir_b")
	pathA := filepath.Join(workdirA, "repo_a")
	pathB := filepath.Join(workdirB, "repo_b")

	s.initGitRepo(pathA)
	s.initGitRepo(pathB)

	// Daemon is stopped, init with workdir A
	r := s.RunInit(workdirA)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select * from repositories")
	require.NoError(r.Error, r.Combined())

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
+---------------+
`)
	require.Contains(r.Stdout(), expected)

	// Daemon is running, calling init with a different workdir should
	// restart gitbase correctly
	r = s.RunInit(workdirB)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select * from repositories")
	require.NoError(r.Error, r.Combined())

	expected = sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_b        |
+---------------+
`)
	require.Contains(r.Stdout(), expected)
}

func (s *InitTestSuite) TestRefreshWorkdir() {
	require := s.Require()

	// Create a with a repo
	workdir := filepath.Join(s.TestDir, "workdir")
	pathA := filepath.Join(workdir, "repo_a")
	pathB := filepath.Join(workdir, "repo_b")

	s.initGitRepo(pathA)

	// Daemon is stopped, init with repo A only
	r := s.RunInit(workdir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select * from repositories")
	require.NoError(r.Error, r.Combined())

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
+---------------+
`)
	require.Contains(r.Stdout(), expected)

	// Init the second git repo
	s.initGitRepo(pathB)

	// Daemon is running, calling init with the same workdir should
	// restart gitbase correctly and see the new repo B
	r = s.RunInit(workdir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select * from repositories order by repository_id")
	require.NoError(r.Error, r.Combined())

	expected = sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
| repo_b        |
+---------------+
`)
	require.Contains(r.Stdout(), expected)
}
