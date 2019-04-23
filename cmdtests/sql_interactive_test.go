// +build integration,!windows

package cmdtests_test

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kr/pty"
	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/require"
)

type interactiveREPLExitMethod int

const (
	exitCmd interactiveREPLExitMethod = iota
	exitCtrlD
)

type interativeREPL struct {
	bin string
}

func (r *interativeREPL) start() (*exec.Cmd, io.Writer, <-chan string, error) {
	// cannot use `icmd` here, please see: https://github.com/gotestyourself/gotest.tools/issues/151
	command := exec.Command(r.bin, "sql")

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

func (r *interativeREPL) query(in io.Writer, query string, out <-chan string) string {
	io.WriteString(in, query)

	var res strings.Builder
	for c := range out {
		if strings.HasPrefix(c, "Empty set") {
			return ""
		}

		res.WriteString(c + "\r\n")
		if r.containsSQLOutput(res.String()) {
			break
		}
	}

	return res.String()
}

func (r *interativeREPL) exitAndWait(exitMethod interactiveREPLExitMethod, timeout time.Duration, in io.Writer, out <-chan string) error {
	switch exitMethod {
	case exitCmd:
		io.WriteString(in, "exit;\n")
	case exitCtrlD:
		io.WriteString(in, string('\004'))
	}

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
func (r *interativeREPL) containsSQLOutput(out string) bool {
	sep := regexp.MustCompile(`\+-+\+`)
	matches := sep.FindAllStringIndex(out, -1)
	return len(matches) == 3
}

func (s *SQLREPLTestSuite) TestInteractiveREPL() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Testing interactive REPL on Windows is not supported")
	}

	testCasesNames := []string{"exit with 'exit' command", "exit with 'Ctrl-D'"}
	for i, em := range []interactiveREPLExitMethod{exitCmd, exitCtrlD} {
		s.T().Run(testCasesNames[i], func(t *testing.T) {
			require := require.New(t)

			repl := &interativeREPL{bin: s.Bin()}

			command, in, out, err := repl.start()
			require.NoError(err)

			res := repl.query(in, "show tables;\n", out)
			require.Contains(res, showTablesOutput)

			res = repl.query(in, "describe table repositories;\n", out)
			require.Contains(res, showRepoTableDescOutput)

			require.NoError(repl.exitAndWait(em, 10*time.Second, in, out))

			require.NoError(command.Wait())
		})
	}
}
