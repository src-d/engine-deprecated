// +build integration

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

func (s *IntegrationSuite) Bin() string {
	return srcdBin
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

type ChannelWriter struct {
	ch chan string
}

func NewChannelWriter(ch chan string) *ChannelWriter {
	return &ChannelWriter{ch: ch}
}

func (cr *ChannelWriter) Write(b []byte) (int, error) {
	cr.ch <- string(b)
	return len(b), nil
}

var newLineFormatter = regexp.MustCompile(`(\r\n|\r|\n)`)

func normalizeNewLine(s string) string {
	return newLineFormatter.ReplaceAllString(s, "\n")
}

// StreamLinifier is useful when we have a stream of messages, where each message
// can contain multiple lines, and we want to transform it into a stream of messages,
// where each message is a single line.
// Example:
//   - input: "foo", "bar\nbaz", "qux\nquux\n"
//   - output: "foo", "bar", "baz", "qux", "quux"
//
// This transformation is done through the `Linify` method that reads the input from
// the channel passed as argument and writes the output into the returned channel.
//
// Corner case:
// given the input message "foo\nbar\baz", the lines "foo" and "bar" are written to
// the output channel ASAP, but notice that it's not possible to do the same for
// "baz" which is then marked as *pending*.
// That's because it doesn't end with a new line. In fact, two cases may hold with
// the following message:
//   1. the following message starts with a new line, let's say "\nqux\n",
//   2. the following message doesn't start with a new line, let'say "qux\n".
//
// In the first case, "baz" can be written to the output channel, but in the second
// case, "qux" is the continuation of the same line of "baz", so "bazqux" is the
// message to be written.
// To avoid losing to write the last line, if there's a pending line and and
// an amount of time equal to `newLineTimeout` elapses, then we consider it
// as a completed line and we write the message to the output channel.
type StreamLinifier struct {
	newLineTimeout time.Duration
	pending        string
}

// NewStreamLinifier returns a `StreamLinifier` configure with a given timeout
func NewStreamLinifier(timeout time.Duration) *StreamLinifier {
	return &StreamLinifier{newLineTimeout: timeout}
}

// Linify returns a channel to read lines from.
// Messages coming from `in` containing multiple newlines (`(\r\n|\r|\n)`), will
// be sent to the returned channel as multiple messages, one per line.
func (sl *StreamLinifier) Linify(in chan string) chan string {
	out := make(chan string)

	go func() {
		for {
			select {
			case <-time.After(sl.newLineTimeout):
				if sl.pending != "" {
					out <- sl.pending
					sl.pending = ""
				}
			case s, ok := <-in:
				if !ok {
					close(out)
					return
				}

				lines := strings.Split(sl.pending+normalizeNewLine(s), "\n")
				sl.pending = ""

				for i, l := range lines {
					if i == len(lines) && l != "" {
						sl.pending = l
						break
					}
					out <- l
				}
			}
		}
	}()

	return out
}
