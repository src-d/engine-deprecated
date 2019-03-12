// +build integration

package cmd

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	cmdtest "github.com/src-d/engine/cmd/test-utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SQLTestSuite struct {
	cmdtest.IntegrationSuite
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
	s.RunStop(context.Background())
	os.RemoveAll(s.testDir)
}

const showTablesOutput = `+--------------+
|    TABLE     |
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
`

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

	expected := `+---------------+
| REPOSITORY ID |
+---------------+
| reponame      |
+---------------+
`
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
			err:   "SQL query failed: Error 1105: unknown error: syntax error at position",
		},
		{
			query: "show tables; show tables",
			err:   "SQL query failed: Error 1105: unknown error: syntax error at position",
		},
		{
			query: "select from repositories",
			err:   "SQL query failed: Error 1105: unknown error: syntax error at position",
		},
		{
			query: "select * from nope",
			err:   "SQL query failed: Error 1105: unknown error: table not found: nope",
		},
		{
			query: "insert into repositories values ('myrepo')",
			err:   "SQL query failed: Error 1105: unknown error: table doesn't support INSERT INTO",
		},
		{
			query: "select nope from repositories",
			err:   `SQL query failed: Error 1105: unknown error: column "nope" could not be found in any table in scope`,
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

	expected := `+--------------+
|    TABLE     |
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
|     NAME      | TYPE |
+---------------+------+
| repository_id | TEXT |
+---------------+------+`

	require.Contains(out.String(), expected)
}
