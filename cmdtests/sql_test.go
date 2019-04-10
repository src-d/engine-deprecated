// +build integration

package cmdtests_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kr/pty"
	"github.com/src-d/engine/cmdtests"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gotest.tools/icmd"
)

var showTablesOutput = sqlOutput(`+--------------+
| Table        |
+--------------+
| blobs        |
| commit_blobs |
| commit_files |
| commit_trees |
| commits      |
| files        |
| ref_commits  |
| refs         |
| remotes      |
| repositories |
| tree_entries |
+--------------+
`)

var showRepoTableDescOutput = sqlOutput(`+---------------+------+
| name          | type |
+---------------+------+
| repository_id | TEXT |
+---------------+------+
`)

type SQLREPLTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
	testDir string
}

func TestSQLREPLTestSuite(t *testing.T) {
	s := SQLREPLTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *SQLREPLTestSuite) TestREPL() {
	// When it is not in a terminal, the command reads stdin and exits.
	// You can see it running:
	//     $ echo "show tables;" | ./srcd sql
	// So this test does not really interacts with the REPL prompt, but we still
	// test that the code that processes each read line is working as expected.

	require := s.Require()

	input := "show tables;\n" + "describe table repositories;\n"
	r := s.RunCmd("sql", nil, icmd.WithStdin(strings.NewReader(input)))
	require.NoError(r.Error, r.Combined())
	require.Contains(r.Combined(), showTablesOutput)
	require.Contains(r.Combined(), showRepoTableDescOutput)
}

func (s *SQLREPLTestSuite) TestInteractiveREPL() {
	if runtime.GOOS == "windows" {
		s.T().Skip("Testing interactive REPL on Windows is not supported")
	}

	require := s.Require()

	command, in, out, err := s.runInteractiveRepl()
	require.NoError(err)

	res := s.runInteractiveQuery(in, "show tables;\n", out)
	require.Contains(res, showTablesOutput)

	res = s.runInteractiveQuery(in, "describe table repositories;\n", out)
	require.Contains(res, showRepoTableDescOutput)

	require.NoError(s.exitInteractiveAndWait(10*time.Second, in, out))
	require.NoError(s.waitMysqlCliContainerStopped(10, 1*time.Second))

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
		if strings.HasPrefix(s, "mysql>") {
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

func (s *SQLREPLTestSuite) waitMysqlCliContainerStopped(retries int, retryTimeout time.Duration) error {
	for i := 0; i < retries; i++ {
		running, err := docker.IsRunning(components.MysqlCli.Name, "")
		if !running {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(retryTimeout)
	}

	return fmt.Errorf("maximum number of retries (%d) reached while waiting to stop container", retries)
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

type SQLTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestSQLTestSuite(t *testing.T) {
	s := SQLTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *SQLTestSuite) TestInit() {
	require := s.Require()

	repoPath := filepath.Join(s.TestDir, "reponame")
	err := os.Mkdir(repoPath, os.ModePerm)
	require.NoError(err)

	cmd := exec.Command("git", "init", repoPath)
	err = cmd.Run()
	require.NoError(err)

	r := s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "select * from repositories")
	require.NoError(r.Error, r.Combined())

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| reponame      |
+---------------+
`)
	require.Contains(r.Stdout(), expected)
}

func (s *SQLTestSuite) TestNoInit() {
	require := s.Require()

	r := s.RunCommand("sql", "show tables")
	require.NoError(r.Error, r.Combined())

	require.Contains(r.Stdout(), showTablesOutput)
}

func (s *SQLTestSuite) TestValidQueries() {
	testCases := []string{
		"show tables",
		"SHOW TABLES",
		"show tables;",
		"SHOW TABLES;",
		" show tables ; ",
		" SHOW TABLES ; ",
		"	show tables	;	",
		"	SHOW TABLES	;	",
		"/* comment */ show tables;",
		`/* multi line
			comment */
			show tables;`,
	}

	for _, query := range testCases {
		s.T().Run(query, func(t *testing.T) {
			assert := assert.New(t)

			r := s.RunCommand("sql", query)
			assert.NoError(r.Error, r.Combined())

			assert.Contains(r.Stdout(), showTablesOutput)
		})
	}
}

func (s *SQLTestSuite) TestWrongQuery() {
	testCases := []struct {
		query string
		err   string
	}{
		{
			query: "show",
			err:   "ERROR 1105 (HY000) at line 1: unknown error: syntax error at position",
		},
		{
			query: "select from repositories",
			err:   "ERROR 1105 (HY000) at line 1: unknown error: syntax error at position",
		},
		{
			query: "select * from nope",
			err:   "ERROR 1105 (HY000) at line 1: unknown error: table not found: nope",
		},
		{
			query: "insert into repositories values ('myrepo')",
			err:   "ERROR 1105 (HY000) at line 1: unknown error: table doesn't support INSERT INTO",
		},
		{
			query: "select nope from repositories",
			err:   `ERROR 1105 (HY000) at line 1: unknown error: column "nope" could not be found in any table in scope`,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.query, func(t *testing.T) {
			assert := assert.New(t)

			r := s.RunCommand("sql", tc.query)
			assert.Error(r.Error)

			assert.Contains(r.Stdout(), tc.err)
		})
	}
}

func (s *SQLTestSuite) TestIndexesWorkdirChange() {
	require := s.Require()

	// use engine repo itself to avoid cloning anything
	wd, err := os.Getwd()
	require.NoError(err)
	wd = filepath.ToSlash(wd)
	enginePath := path.Join(wd, "..")

	// workdir 1
	r := s.RunInit(enginePath)
	require.NoError(r.Error, r.Combined())

	r = s.RunCommand("sql", "CREATE INDEX repo_idx ON repositories USING pilosa (repository_id)")
	require.NoError(err, r.Stdout())

	s.testQueryWithIndex(require, "repos", true)

	// workdir 2
	repoPath := filepath.Join(s.TestDir, "reponame")
	err = os.Mkdir(repoPath, os.ModePerm)
	require.NoError(err)

	cmd := exec.Command("git", "init", repoPath)
	err = cmd.Run()
	require.NoError(err)

	r = s.RunInit(s.TestDir)
	require.NoError(r.Error, r.Combined())

	s.testQueryWithIndex(require, "reponame", false)

	// back to workdir 1
	r = s.RunInit(enginePath)
	require.NoError(r.Error, r.Combined())

	// wait for gitbase to be ready
	r = s.RunCommand("sql", "select 1")
	require.NoError(r.Error, r.Combined())

	s.testQueryWithIndex(require, "repos", true)
}

func (s *SQLTestSuite) testQueryWithIndex(require *require.Assertions, repo string, hasIndex bool) {
	if hasIndex {
		require.True(cmdtests.IndexIsVisible(s, "repositories", "repo_idx"))
	}

	r := s.RunCommand("sql", "EXPLAIN FORMAT=TREE select * from repositories WHERE repository_id='"+repo+"'")
	require.NoError(r.Error, r.Combined())
	if hasIndex {
		require.Contains(r.Stdout(), "Indexes")
	} else {
		require.NotContains(r.Stdout(), "Indexes")
	}

	r = s.RunCommand("sql", "select * from repositories WHERE repository_id='"+repo+"'")
	require.NoError(r.Error, r.Combined())
	require.Contains(r.Stdout(), repo)
}

func sqlOutput(v string) string {
	return strings.Replace(v, "\n", "\r\n", -1)
}
