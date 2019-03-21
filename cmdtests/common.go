// +build integration

package cmdtests

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/src-d/engine/docker"
	"github.com/stretchr/testify/suite"
)

// TODO (carlosms) this could be build/bin, workaround for https://github.com/src-d/ci/issues/97
var srcdBin = fmt.Sprintf("../build/engine_%s_%s/srcd", runtime.GOOS, runtime.GOARCH)

func init() {
	if os.Getenv("SRCD_BIN") != "" {
		srcdBin = os.Getenv("SRCD_BIN")
	}
}

type IntegrationSuite struct {
	suite.Suite
	*Commander
}

func NewIntegrationSuite() IntegrationSuite {
	return IntegrationSuite{Commander: &Commander{bin: srcdBin}}
}

func (s *IntegrationSuite) SetupTest() {
	// make sure previous tests don't affect engine state
	// as long as prune works correctly
	//
	// NB: don't run prune on TearDown to be able to see artifacts of failed test
	r := s.RunCommand("prune")
	s.Require().NoError(r.Error, r.Combined())
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

func NewIntegrationTmpDirSuite() IntegrationTmpDirSuite {
	return IntegrationTmpDirSuite{IntegrationSuite: NewIntegrationSuite()}
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

type RegressionSuite struct {
	suite.Suite
	PrevCmd *Commander
	CurrCmd *Commander
}

func NewRegressionSuite(prevBin, currentBin string) RegressionSuite {
	return RegressionSuite{
		PrevCmd: &Commander{bin: prevBin},
		CurrCmd: &Commander{bin: currentBin},
	}
}

type SQLOutputTable struct {
	Data  map[string][]string
	cols  []string
	rowsN int
}

func (s *SQLOutputTable) RequireEqual(o *SQLOutputTable) error {
	return s.requireEqual(o, false)
}

func (s *SQLOutputTable) RequireStrictlyEqual(o *SQLOutputTable) error {
	return s.requireEqual(o, true)
}

func (s *SQLOutputTable) requireEqual(o *SQLOutputTable, strictEmpty bool) error {
	if !strictEmpty && s.rowsN == 0 && o.rowsN == 0 {
		return nil
	}

	if s.rowsN != o.rowsN {
		return s.diffErr("rows number", s.rowsN, o.rowsN)
	}

	if !reflect.DeepEqual(s.cols, o.cols) {
		return s.diffErr("columns", s.cols, o.cols)
	}

	var thisRows []string
	var otherRows []string

	for i := 0; i < s.rowsN; i++ {
		var thisRow []string
		var otherRow []string

		for _, c := range s.cols {
			thisRow = append(thisRow, s.Data[c][i])
			otherRow = append(otherRow, o.Data[c][i])
		}

		thisRows = append(thisRows, strings.Join(thisRow, "|"))
		otherRows = append(otherRows, strings.Join(otherRow, "|"))
	}

	sort.Strings(thisRows)
	sort.Strings(otherRows)

	eq := reflect.DeepEqual(thisRows, otherRows)
	if eq {
		return nil
	}

	return s.diffErr("rows", thisRows, otherRows)
}

func (s *SQLOutputTable) diffErr(what string, this, other interface{}) error {
	return fmt.Errorf("Different %s:\n- actual:   %v\n- expected: %v",
		what, this, other)
}

func AreSQLOutputStrictlyEqual(s1 string, s2 string) error {
	return ParseSQLOutput(s1).RequireStrictlyEqual(ParseSQLOutput(s2))
}

func AreSQLOutputEqual(s1 string, s2 string) error {
	return ParseSQLOutput(s1).RequireEqual(ParseSQLOutput(s2))
}

var newLineFormatter = regexp.MustCompile(`(\r\n|\r|\n)`)
var lineSepReg = regexp.MustCompile(`^\+[-+]+\+$`)
var lineReg = regexp.MustCompile(`(?:\s+((?:[\w-\s.]+)?)\s+)`)

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

func normalizeColName(s string) string {
	normCol := strings.ToUpper(strings.TrimSpace(s))
	return strings.Replace(
		strings.Replace(normCol, " ", "_", -1),
		"-", "_", -1)
}

func ParseSQLOutput(raw string) *SQLOutputTable {
	splitted := strings.Split(normalizeNewLine(raw), "\n")
	header := false
	body := false
	var cols []string
	fields := make(map[string][]string)
	nRows := 0
	for _, s := range splitted {
		if !header && !body {
			if lineSepReg.MatchString(s) {
				header = true
			}

			continue
		}

		if header {
			if lineSepReg.MatchString(s) {
				header = false
				body = true
				continue
			}

			for _, match := range lineReg.FindAllStringSubmatch(s, -1) {
				cols = append(cols, normalizeColName(match[1]))
			}
		}

		if body {
			if lineSepReg.MatchString(s) {
				break
			}

			nRows++
			for i, match := range lineReg.FindAllStringSubmatch(s, -1) {
				key := cols[i]
				fields[key] = append(fields[key], match[1])
			}
		}
	}

	sort.Strings(cols)
	return &SQLOutputTable{
		Data:  fields,
		cols:  cols,
		rowsN: nRows,
	}
}
