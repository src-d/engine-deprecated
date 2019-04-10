// +build regression

package cmdtests_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/src-d/engine/cmdtests"
	"gotest.tools/icmd"

	"gopkg.in/src-d/regression-core.v0"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/suite"
)

var prevVersion = "latest"
var currVersion = "local:HEAD"

func init() {
	if os.Getenv("PREV_ENGINE_VERSION") != "" {
		prevVersion = os.Getenv("PREV_ENGINE_VERSION")
	}

	if os.Getenv("CURR_ENGINE_VERSION") != "" {
		currVersion = os.Getenv("CURR_ENGINE_VERSION")
	}
}

type GitbaseBackCompTestSuite struct {
	cmdtests.RegressionSuite
	timeout     time.Duration
	testDir     string
	testDataDir string
}

func TestGitbaseBackCompTestSuite(t *testing.T) {
	s := GitbaseBackCompTestSuite{timeout: 10 * time.Minute}
	suite.Run(t, &s)
}

func (s *GitbaseBackCompTestSuite) createRepo(name string) {
	s.T().Helper()
	require := s.Require()

	repoPath := filepath.Join(s.testDir, name)
	err := os.Mkdir(repoPath, os.ModePerm)
	require.NoError(err)

	cmd := exec.Command("git", "init", repoPath)
	err = cmd.Run()
	require.NoError(err)
}

func (s *GitbaseBackCompTestSuite) SetupSuite() {
	token := os.Getenv("GITHUB_TOKEN")

	// This is required because regression-core assumes that the working
	// directory is the root of the project
	s.setupWd()

	config := regression.NewConfig()
	cachePath := "regression-testing-cache"
	config.BinaryCache = filepath.Join(cachePath, "binary")
	config.RepositoriesCache = filepath.Join(cachePath, "repos")

	tool := regression.Tool{
		Name:        "engine",
		BinaryName:  "srcd",
		GitURL:      "https://github.com/src-d/engine",
		ProjectPath: "github.com/src-d/engine",
		BuildSteps: []regression.BuildStep{
			{
				Dir:     "",
				Command: "make",
				Args:    []string{"build", "docker-build"},
			},
		},
	}

	releases := regression.NewReleases("src-d", "engine", token)

	s.PrevCmd = cmdtests.NewCommander(s.getBinary(config, tool, releases, prevVersion))
	s.CurrCmd = cmdtests.NewCommander(s.getBinary(config, tool, releases, currVersion))
}

// setupWd fixes the working directory by setting it to the root of the project.
// This has to be done because regression-core assumes that the working directory
// is the root of the project, but `go test` sets the working directory accordingly
// to the running test.
func (s *GitbaseBackCompTestSuite) setupWd() {
	s.T().Helper()

	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		s.T().Fatalf("cannot change working dir to %s: %v", dir, err)
	}
}

func (s *GitbaseBackCompTestSuite) getBinary(conf regression.Config, tool regression.Tool, releases *regression.Releases, version string) string {
	s.T().Helper()

	bin := regression.NewBinary(conf, tool, version, releases)
	if err := bin.Download(); err != nil {
		s.T().Fatalf("%v", err)
	}

	return bin.Path
}

func (s *GitbaseBackCompTestSuite) SetupTest() {
	testDir, err := ioutil.TempDir("", "gitbase-back-comp-test")
	s.testDir = testDir
	if err != nil {
		s.T().Fatalf("%v", err)
	}

	homedir, err := homedir.Dir()
	if err != nil {
		s.T().Fatalf("unable to get home dir: %v", err)
	}

	// this is needed only for engine < 0.12
	s.testDataDir = filepath.ToSlash(filepath.Join(homedir, ".srcd", "gitbase"))
	os.RemoveAll(s.testDataDir)

	for i := range [3]int{} {
		s.createRepo(fmt.Sprintf("repo-%d", i))
	}
}

func (s *GitbaseBackCompTestSuite) TearDownTest() {
	s.CurrCmd.RunCommand("stop")
	s.PrevCmd.RunCommand("stop")
	os.RemoveAll(s.testDir)
	os.RemoveAll(s.testDataDir)
}

func (s *GitbaseBackCompTestSuite) runSQL(cmd *cmdtests.Commander, query string) *icmd.Result {
	s.T().Helper()

	return cmd.RunCmd("sql", []string{query}, icmd.WithTimeout(s.timeout))
}

func (s *GitbaseBackCompTestSuite) TestRetroCompatibleIndexes() {
	require := s.Require()

	// [previous version] srcd init s.testDir
	r := s.PrevCmd.RunInitWithTimeout(s.testDir, s.timeout)
	require.NoError(r.Error, r.Combined())

	// [previous version] srcd sql "select * from repositories"
	r = s.runSQL(s.PrevCmd, "SELECT * FROM repositories")
	require.NoError(r.Error, r.Combined())

	expected := `+---------------+
| REPOSITORY ID |
+---------------+
| repo-0        |
| repo-1        |
| repo-2        |
+---------------+`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [previous version] srcd sql "SHOW INDEXES"
	r = s.runSQL(s.PrevCmd, "SHOW INDEX FROM repositories")

	require.NoError(r.Error, r.Combined())
	expected = `+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
| TABLE | NON UNIQUE | KEY NAME | SEQ IN INDEX | COLUMN NAME | COLLATION | CARDINALITY | SUB PART | PACKED | NULL | INDEX TYPE | COMMENT | INDEX COMMENT | VISIBLE | EXPRESSION |
+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [previous version] srcd sql "CREATE INDEXES"
	r = s.runSQL(s.PrevCmd, "CREATE INDEX repo_idx ON repositories USING pilosa (repository_id)")
	require.NoError(r.Error, r.Combined())
	// wait a bit for index to be ready
	time.Sleep(1 * time.Second)

	// [previous version] srcd sql "SHOW INDEXES"
	r = s.runSQL(s.PrevCmd, "SHOW INDEX FROM repositories")
	require.NoError(r.Error, r.Combined())
	expected = `+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
|    TABLE     | NON UNIQUE | KEY NAME | SEQ IN INDEX |        COLUMN NAME         | COLLATION | CARDINALITY | SUB PART | PACKED | NULL | INDEX TYPE | COMMENT | INDEX COMMENT | VISIBLE | EXPRESSION |
+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
| repositories |          1 | repo_idx |            1 | repositories.repository_id | NULL      |           0 |        0 | NULL   |      | pilosa     |         |               | YES     | NULL       |
+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [previous version] srcd sql "explain select * from repositories"
	r = s.runSQL(s.PrevCmd, "EXPLAIN FORMAT=tree SELECT * FROM repositories WHERE repository_id='repo-0'")
	require.NoError(r.Error, r.Combined())
	require.Contains(strings.ToUpper(r.Combined()), "INDEXES")
	require.Contains(r.Combined(), "repo_idx")

	// [previous version] srcd stop
	r = s.PrevCmd.RunCommand("stop")
	require.NoError(r.Error, r.Combined())

	// [current version] srcd init s.testDir
	r = s.CurrCmd.RunInitWithTimeout(s.testDir, s.timeout)
	require.NoError(r.Error, r.Combined())

	// [current version] srcd sql "select * from repositories"
	r = s.runSQL(s.CurrCmd, "SELECT * FROM repositories")
	require.NoError(r.Error, r.Combined())

	expected = `+---------------+
| REPOSITORY ID |
+---------------+
| repo-0        |
| repo-1        |
| repo-2        |
+---------------+`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [current version] srcd sql "SHOW INDEXES"
	r = s.runSQL(s.CurrCmd, "SHOW INDEX FROM repositories")
	require.NoError(r.Error, r.Combined())
	expected = `+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
|    TABLE     | NON UNIQUE | KEY NAME | SEQ IN INDEX |        COLUMN NAME         | COLLATION | CARDINALITY | SUB PART | PACKED | NULL | INDEX TYPE | COMMENT | INDEX COMMENT | VISIBLE | EXPRESSION |
+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
| repositories |          1 | repo_idx |            1 | repositories.repository_id | NULL      |           0 |        0 | NULL   |      | pilosa     |         |               | YES     | NULL       |
+--------------+------------+----------+--------------+----------------------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [current version] srcd sql "explain select * from repositories"
	r = s.runSQL(s.CurrCmd, "EXPLAIN FORMAT=tree SELECT * FROM repositories WHERE repository_id='repo-0'")
	require.NoError(r.Error, r.Combined())
	require.Contains(strings.ToUpper(r.Combined()), "INDEXES")
	require.Contains(r.Combined(), "repo_idx")

	// [current version] srcd sql "DROP INDEXES"
	r = s.runSQL(s.CurrCmd, "DROP INDEX repo_idx ON repositories")
	require.NoError(r.Error, r.Combined())

	// [current version] srcd sql "SHOW INDEXES"
	r = s.runSQL(s.CurrCmd, "SHOW INDEX FROM repositories")
	require.NoError(r.Error, r.Combined())
	expected = `+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
| TABLE | NON UNIQUE | KEY NAME | SEQ IN INDEX | COLUMN NAME | COLLATION | CARDINALITY | SUB PART | PACKED | NULL | INDEX TYPE | COMMENT | INDEX COMMENT | VISIBLE | EXPRESSION |
+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
+-------+------------+----------+--------------+-------------+-----------+-------------+----------+--------+------+------------+---------+---------------+---------+------------+
`
	require.NoError(cmdtests.AreSQLOutputEqual(r.Combined(), expected))

	// [current version] srcd sql "explain select * from repositories"
	r = s.runSQL(s.CurrCmd, "EXPLAIN FORMAT=tree SELECT * FROM repositories")
	require.NoError(r.Error, r.Combined())
	require.NotContains(strings.ToUpper(r.Combined()), "INDEXES")
	require.NotContains(r.Combined(), "repo_idx")
}
