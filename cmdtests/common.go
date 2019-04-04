package cmdtests

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/src-d/engine/docker"
	"github.com/stretchr/testify/suite"
	"gotest.tools/icmd"
)

// TODO (carlosms) this could be build/bin, workaround for https://github.com/src-d/ci/issues/97
var srcdBin = fmt.Sprintf("../build/engine_%s_%s/srcd", runtime.GOOS, runtime.GOARCH)
var configFile = "../integration-testing-config.yaml"

func init() {
	if os.Getenv("SRCD_BIN") != "" {
		srcdBin = os.Getenv("SRCD_BIN")
	}
}

type IntegrationSuite struct {
	suite.Suite
}

func (s *IntegrationSuite) SetupTest() {
	// make sure previous tests don't affect engine state
	// as long as prune works correctly
	//
	// NB: don't run prune on TearDown to be able to see artifacts of failed test
	r := s.RunCommand("prune")
	s.Require().NoError(r.Error, r.Combined())
}

func (s *IntegrationSuite) RunCmd(cmd string, args []string, cmdOperators ...icmd.CmdOp) *icmd.Result {
	args = append([]string{cmd}, args...)
	return icmd.RunCmd(icmd.Command(srcdBin, args...), cmdOperators...)
}

func (s *IntegrationSuite) RunCommand(cmd string, args ...string) *icmd.Result {
	return s.RunCmd(cmd, args)
}

func (s *IntegrationSuite) StartCommand(cmd string, args []string, cmdOperators ...icmd.CmdOp) *icmd.Result {
	args = append([]string{cmd}, args...)
	return icmd.StartCmd(icmd.Command(srcdBin, args...))
}

func (s *IntegrationSuite) Wait(timeout time.Duration, r *icmd.Result) *icmd.Result {
	return icmd.WaitOnCmd(timeout, r)
}

// RunInit runs srcd init with workdir and custom config for integration tests
func (s *IntegrationSuite) RunInit(workdir string) *icmd.Result {
	return s.RunCommand("init", workdir, "--config", configFile)
}

var logMsgRegex = regexp.MustCompile(`.*msg="(.+?[^\\])"`)

func (s *IntegrationSuite) ParseLogMessages(memLog string) []string {
	var logMessages []string
	for _, line := range strings.Split(memLog, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		match := logMsgRegex.FindStringSubmatch(line)
		if len(match) > 1 {
			logMessages = append(logMessages, match[1])
		}
	}

	return logMessages
}

func (s *IntegrationSuite) AllStopped() {
	s.T().Helper()
	require := s.Require()

	containers := []string{
		"srcd-cli-bblfshd",
		"srcd-cli-bblfsh-web",
		"srcd-cli-daemon",
		"srcd-cli-gitbase-web",
		"srcd-cli-gitbase",
	}

	for _, name := range containers {
		r, err := docker.IsRunning(name, "")
		require.NoError(err)

		require.Falsef(r, "Component %s should not be running", name)
	}
}

type IntegrationTmpDirSuite struct {
	IntegrationSuite
	TestDir string
}

func (s *IntegrationTmpDirSuite) SetupTest() {
	s.IntegrationSuite.SetupTest()

	var err error
	s.TestDir, err = ioutil.TempDir("", strings.Replace(s.T().Name(), "/", "_", -1))
	if err != nil {
		log.Fatal(err)
	}
}

func (s *IntegrationTmpDirSuite) TearDownTest() {
	os.RemoveAll(s.TestDir)
}
