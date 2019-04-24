// +build integration

package cmdtests_test

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/src-d/engine/cmdtests"
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
		require.True(cmdtests.IndexIsVisible(s, s.Commander, "repositories", "repo_idx"))
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
