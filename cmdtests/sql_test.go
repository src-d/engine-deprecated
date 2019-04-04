// +build integration

package cmdtests_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SQLTestSuite struct {
	cmdtests.IntegrationSuite
	testDir string
}

func TestSQLTestSuite(t *testing.T) {
	s := SQLTestSuite{}
	suite.Run(t, &s)
}

func (s *SQLTestSuite) SetupTest() {
	var err error
	s.testDir, err = ioutil.TempDir("", "sql-test")
	if err != nil {
		log.Fatal(err)
	}
}

func (s *SQLTestSuite) TearDownTest() {
	s.RunCommand(context.Background(), "prune")
	os.RemoveAll(s.testDir)
}

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

func (s *SQLTestSuite) TestInit() {
	require := s.Require()

	repoPath := filepath.Join(s.testDir, "reponame")
	err := os.Mkdir(repoPath, os.ModePerm)
	require.NoError(err)

	cmd := exec.Command("git", "init", repoPath)
	err = cmd.Run()
	require.NoError(err)

	_, err = s.RunInit(context.TODO(), s.testDir)
	require.NoError(err)

	buf, err := s.RunSQL(context.TODO(), "select * from repositories")
	require.NoError(err)

	expected := sqlOutput(`+---------------+
| repository_id |
+---------------+
| reponame      |
+---------------+
`)
	require.Contains(buf.String(), expected)
}

func (s *SQLTestSuite) TestNoInit() {
	require := s.Require()

	buf, err := s.RunSQL(context.TODO(), "show tables")
	require.NoError(err)

	require.Contains(buf.String(), showTablesOutput)
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

			buf, err := s.RunSQL(context.TODO(), query)
			assert.NoError(err)

			assert.Contains(buf.String(), showTablesOutput)
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

			buf, err := s.RunSQL(context.TODO(), tc.query)
			assert.Error(err)

			assert.Contains(buf.String(), tc.err)
		})
	}
}

func (s *SQLTestSuite) TestREPL() {
	// When it is not in a terminal, the command reads stdin and exits.
	// You can see it running:
	//     $ echo "show tables;" | ./srcd sql
	// So this test does not really interacts with the REPL prompt, but we still
	// test that the code that processes each read line is working as expected.

	require := s.Require()

	var out, in bytes.Buffer

	command := s.CommandContext(context.TODO(), "sql")
	command.Stdout = &out
	command.Stderr = &out
	command.Stdin = &in

	io.WriteString(&in, "show tables;\n")
	io.WriteString(&in, "describe table repositories;\n")

	err := command.Start()
	require.NoError(err)

	err = command.Wait()
	require.NoError(err)

	expected := sqlOutput(`+--------------+
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
+---------------+------+
| name          | type |
+---------------+------+
| repository_id | TEXT |
+---------------+------+`)

	require.Contains(out.String(), expected)
}

func (s *SQLTestSuite) TestIndexesWorkdirChange() {
	require := s.Require()

	// use engine repo itself to avoid cloning anything
	wd, err := os.Getwd()
	require.NoError(err)
	wd = filepath.ToSlash(wd)
	enginePath := path.Join(wd, "..")

	// workdir 1
	_, err = s.RunInit(context.TODO(), enginePath)
	require.NoError(err)

	buf, err := s.RunSQL(context.TODO(), "CREATE INDEX repo_idx ON repositories USING pilosa (repository_id)")
	require.NoError(err, buf.String())

	time.Sleep(1 * time.Second) // wait for index to be built

	s.testQueryWithIndex(require, "repos", true)

	// workdir 2
	repoPath := filepath.Join(s.testDir, "reponame")
	err = os.Mkdir(repoPath, os.ModePerm)
	require.NoError(err)

	cmd := exec.Command("git", "init", repoPath)
	err = cmd.Run()
	require.NoError(err)

	_, err = s.RunInit(context.TODO(), s.testDir)
	require.NoError(err)

	s.testQueryWithIndex(require, "reponame", false)

	// back to workdir 1
	_, err = s.RunInit(context.TODO(), enginePath)
	require.NoError(err)

	// wait for gitbase to be ready
	buf, err = s.RunSQL(context.TODO(), "select 1")
	require.NoError(err, buf.String())
	// wait for gitbase to load index
	time.Sleep(1 * time.Second)

	s.testQueryWithIndex(require, "repos", true)
}

func (s *SQLTestSuite) testQueryWithIndex(require *require.Assertions, repo string, hasIndex bool) {
	buf, err := s.RunSQL(context.TODO(), "SHOW INDEX FROM repositories")
	require.NoError(err, buf.String())

	if hasIndex {
		// parse result and check that correct index was built and it is visiable
		indexLine := strings.Split(buf.String(), "\n")[3]
		expected := `repositories.repository_id`
		require.Contains(indexLine, expected)
		visibleValue := strings.TrimSpace(strings.Split(indexLine, "|")[14])
		require.Equal("YES", visibleValue)
	}

	buf, err = s.RunSQL(context.TODO(), "EXPLAIN FORMAT=TREE select * from repositories WHERE repository_id='"+repo+"'")
	require.NoError(err, buf.String())
	if hasIndex {
		require.Contains(buf.String(), "Indexes")
	} else {
		require.NotContains(buf.String(), "Indexes")
	}

	buf, err = s.RunSQL(context.TODO(), "select * from repositories WHERE repository_id='"+repo+"'")
	require.NoError(err)
	require.Contains(buf.String(), repo)
}

func sqlOutput(v string) string {
	return strings.Replace(v, "\n", "\r\n", -1)
}
