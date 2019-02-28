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
	"github.com/stretchr/testify/suite"
)

var srcdBin = fmt.Sprintf("../../../build-integration/engine_%s_%s/srcd", runtime.GOOS, runtime.GOARCH)

type IntegrationSuite struct {
	suite.Suite
}

func init() {
	if os.Getenv("SRCD_BIN") != "" {
		srcdBin = os.Getenv("SRCD_BIN")
	}
}

func (s *IntegrationSuite) runCommand(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error) {
	args = append([]string{cmd}, args...)

	var out bytes.Buffer

	command := exec.CommandContext(ctx, srcdBin, args...)
	command.Stdout = &out
	command.Stderr = &out

	return &out, command.Run()
}

var logMsgRegex = regexp.MustCompile(`.*msg="(.+?[^\\])"`)

func (s *IntegrationSuite) ParseLogMessages(memLog *bytes.Buffer) []string {
	var logMessages []string
	for _, line := range strings.Split(memLog.String(), "\n") {
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

func (s *IntegrationSuite) RunInit(ctx context.Context, workdir string) (*bytes.Buffer, error) {
	return s.runCommand(ctx, "init", workdir)
}

func (s *IntegrationSuite) RunSQL(ctx context.Context, query string) (*bytes.Buffer, error) {
	return s.runCommand(ctx, "sql", query)
}

func (s *IntegrationSuite) RunStop(ctx context.Context) (*bytes.Buffer, error) {
	return s.runCommand(ctx, "stop")
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
