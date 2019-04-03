// +build integration

package cmdtests_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	itt := InitTestSuite{timeout: 1 * time.Minute}
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

	workdir, _ := os.Getwd()
	expectedMsg := [2]string{
		logMsg("starting daemon with working directory: %s", workdir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) TestWithValidWorkdir() {
	require := s.Require()

	buf, err := s.runInit(s.validWorkDir)
	require.NoError(err)

	actualMsg := s.getLogMessages(buf)

	expectedMsg := [2]string{
		logMsg("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
	}
}

func (s *InitTestSuite) TestWithInvalidWorkdir() {
	require := s.Require()

	buf, err := s.runInit(s.invalidWorkDir)
	require.Error(err)

	actualMsg := s.getLogMessages(buf)
	require.Equal(1, len(actualMsg))

	expectedMsg := [1]string{
		fmt.Sprintf("path '%s' is not a valid working directory", s.invalidWorkDir),
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

	expectedMsg := [3]string{
		logMsg("removing container %s", components.Daemon.Name),
		logMsg("starting daemon with working directory: %s", s.validWorkDir),
		"daemon started",
	}

	for _, exp := range expectedMsg {
		require.Contains(actualMsg, exp)
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

	expectedMsg := [5]string{
		logMsg("removing container %s", components.Bblfshd.Name),
		logMsg("removing container %s", components.Daemon.Name),
		logMsg("removing container %s", components.Gitbase.Name),
		logMsg("starting daemon with working directory: %s", s.validWorkDir),
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
	out, err := s.runInit(workdirA)
	require.NoError(err, out.String())

	out, err = s.RunSQL(context.TODO(), "select * from repositories")
	require.NoError(err, out.String())

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
+---------------+
`)
	require.Contains(out.String(), expected)

	// Daemon is running, calling init with a different workdir should
	// restart gitbase correctly
	out, err = s.runInit(workdirB)
	require.NoError(err, out.String())

	out, err = s.RunSQL(context.TODO(), "select * from repositories")
	require.NoError(err, out.String())

	expected = sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_b        |
+---------------+
`)
	require.Contains(out.String(), expected)
}

func (s *InitTestSuite) TestRefreshWorkdir() {
	require := s.Require()

	// Create a with a repo
	workdir := filepath.Join(s.TestDir, "workdir")
	pathA := filepath.Join(workdir, "repo_a")
	pathB := filepath.Join(workdir, "repo_b")

	s.initGitRepo(pathA)

	// Daemon is stopped, init with repo A only
	out, err := s.runInit(workdir)
	require.NoError(err, out.String())

	out, err = s.RunSQL(context.TODO(), "select * from repositories")
	require.NoError(err, out.String())

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
+---------------+
`)
	require.Contains(out.String(), expected)

	// Init the second git repo
	s.initGitRepo(pathB)

	// Daemon is running, calling init with the same workdir should
	// restart gitbase correctly and see the new repo B
	out, err = s.runInit(workdir)
	require.NoError(err, out.String())

	out, err = s.RunSQL(context.TODO(), "select * from repositories order by repository_id")
	require.NoError(err, out.String())

	expected = sqlOutput(`+---------------+
| repository_id |
+---------------+
| repo_a        |
| repo_b        |
+---------------+
`)
	require.Contains(out.String(), expected)
}

// formats string the same way as it is printed by logger
func logMsg(format string, args ...string) string {
	escapedArgs := make([]interface{}, len(args))
	for i, v := range args {
		escapedArgs[i] = strings.Replace(v, `\`, `\\`, -1)
	}

	return fmt.Sprintf(format, escapedArgs...)
}
