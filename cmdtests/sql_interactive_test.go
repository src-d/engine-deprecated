// +build integration,!windows

package cmdtests_test

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/kr/pty"
	"github.com/src-d/engine/cmdtests"
)

func (s *SQLREPLTestSuite) TestInteractiveREPL() {
	require := s.Require()

	command, in, out, err := s.runInteractiveRepl()
	require.NoError(err)

	res := s.runInteractiveQuery(in, "show tables;\n", out)
	require.Contains(res, showTablesOutput)

	res = s.runInteractiveQuery(in, "describe table repositories;\n", out)
	require.Contains(res, showRepoTableDescOutput)

	require.NoError(s.exitInteractiveAndWait(10*time.Second, in, out))

	command.Wait()
}

func (s *SQLREPLTestSuite) runInteractiveRepl() (*exec.Cmd, io.Writer, <-chan string, error) {
	s.T().Helper()

	// cannot use `icmd` here, please see: https://github.com/gotestyourself/gotest.tools/issues/151
	command := exec.Command(s.Bin(), "sql")

	ch := make(chan string)
	cr := cmdtests.NewChannelWriter(ch)

	command.Stdout = cr
	command.Stderr = cr

	in, err := pty.Start(command)
	if err != nil {
		panic(err)
	}

	linifier := cmdtests.NewStreamLinifier(1 * time.Second)
	out := linifier.Linify(ch)
	for s := range out {
		if strings.Contains(s, "MySQL [(none)]>") {
			return command, in, out, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("Mysql cli prompt never started")
}

func (s *SQLREPLTestSuite) runInteractiveQuery(in io.Writer, query string, out <-chan string) string {
	io.WriteString(in, query)

	var res strings.Builder
	for c := range out {
		if strings.HasPrefix(c, "Empty set") {
			return ""
		}

		res.WriteString(c + "\r\n")
		if s.containsSQLOutput(res.String()) {
			break
		}
	}

	return res.String()
}

func (s *SQLREPLTestSuite) exitInteractiveAndWait(timeout time.Duration, in io.Writer, out <-chan string) error {
	io.WriteString(in, "exit;\n")

	done := make(chan struct{})
	go func() {
		for c := range out {
			if strings.Contains(c, "Bye") {
				done <- struct{}{}
				// don't return in order to consume all output and let the process exit
			}
		}
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout of %v elapsed while waiting to exit", timeout)
	}
}

// containsSQLOutput returns `true` if the given string is a SQL output table.
// To detect whether the `out` is a SQL output table, this checks that there
// are exactly 3 separators matching this regex ``\+-+\+`.
// In fact an example of SQL output is the following:
//
//     +--------------+  <-- first separator
//     | Table        |
//     +--------------+  <-- second separator
//     | blobs        |
//     | commit_blobs |
//     | commit_files |
//     | commit_trees |
//     | commits      |
//     | files        |
//     | ref_commits  |
//     | refs         |
//     | remotes      |
//     | repositories |
//     | tree_entries |
//     +--------------+  <-- third separator
//
func (s *SQLREPLTestSuite) containsSQLOutput(out string) bool {
	sep := regexp.MustCompile(`\+-+\+`)
	matches := sep.FindAllStringIndex(out, -1)
	return len(matches) == 3
}
