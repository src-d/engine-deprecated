package cmdtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/src-d/engine/docker"
	"github.com/stretchr/testify/suite"
)

// TODO (carlosms) this could be build/bin, workaround for https://github.com/src-d/ci/issues/97
var srcdBin = fmt.Sprintf("../../../build/engine_%s_%s/srcd", runtime.GOOS, runtime.GOARCH)
var configFile = "../../../integration-testing-config.yaml"

type IntegrationSuite struct {
	suite.Suite
}

func init() {
	if os.Getenv("SRCD_BIN") != "" {
		srcdBin = os.Getenv("SRCD_BIN")
	}
}

func (s *IntegrationSuite) CommandContext(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	args = append([]string{cmd}, args...)
	return exec.CommandContext(ctx, srcdBin, args...)
}

func (s *IntegrationSuite) RunCommand(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	command := s.CommandContext(ctx, cmd, args...)
	command.Stdout = &out
	command.Stderr = &out

	return &out, command.Run()
}

var logMsgRegex = regexp.MustCompile(`.*msg="(.+?[^\\])"`)

func (s *IntegrationSuite) ParseLogMessages(memLog *bytes.Buffer) []string {
	// In case of error the usage of the command is printed and after that
	// the error. Given that it is not handled by logrus, it must be parsed
	// manually. The logged usage is not added to the log messages.
	var logMessages []string
	parsingUsage := false
	parsingFlags := false
	finishedUsage := false
	for _, line := range strings.Split(memLog.String(), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if parsingFlags {
			if strings.HasPrefix(line, "-") || line == "Global Flags:" {
				continue
			}

			parsingUsage = false
			parsingFlags = false
			finishedUsage = true
		}

		if finishedUsage {
			logMessages = append(logMessages, line)
			continue
		}

		if parsingUsage {
			if line == "Flags:" {
				parsingFlags = true
				continue
			}
		}

		if line == "Usage:" {
			parsingUsage = true
			continue
		}

		match := logMsgRegex.FindStringSubmatch(line)
		if len(match) > 1 {
			logMessages = append(logMessages, match[1])
		}
	}

	return logMessages
}

func (s *IntegrationSuite) RunInit(ctx context.Context, workdir string) (*bytes.Buffer, error) {
	return s.RunCommand(ctx, "init", workdir, "--config", configFile)
}

func (s *IntegrationSuite) RunSQL(ctx context.Context, query string) (*bytes.Buffer, error) {
	return s.RunCommand(ctx, "sql", query)
}

func (s *IntegrationSuite) RunStop(ctx context.Context) (*bytes.Buffer, error) {
	return s.RunCommand(ctx, "stop")
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

type LogMessage struct {
	Msg   string
	Time  string
	Level string
}

func TraceLogMessages(fn func(), memLog *bytes.Buffer) []LogMessage {
	logrus.SetOutput(memLog)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	fn()

	var result []LogMessage
	if memLog.Len() == 0 {
		return result
	}

	dec := json.NewDecoder(strings.NewReader(memLog.String()))
	for {
		var i LogMessage
		err := dec.Decode(&i)
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		result = append(result, i)
	}

	return result
}
