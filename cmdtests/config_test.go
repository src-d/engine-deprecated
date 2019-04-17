// +build integration

package cmdtests_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/src-d/engine/cmd/srcd/config"
	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	cmdtests.IntegrationTmpDirSuite
}

func TestConfigTestSuite(t *testing.T) {
	s := ConfigTestSuite{IntegrationTmpDirSuite: cmdtests.NewIntegrationTmpDirSuite()}
	suite.Run(t, &s)
}

func (s *ConfigTestSuite) SetupSuite() {
	// Instead of doing complicated things to test an interactive text editor,
	// using 'cat' we can capture the output
	os.Setenv("EDITOR", "cat")
}

func (s *ConfigTestSuite) SetupTest() {
	s.IntegrationTmpDirSuite.SetupTest()

	// To test $HOME/.srcd/config.yml without breaking any existing installation
	tmpHome := filepath.Join(s.TestDir, "home")
	os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)
}

func (s *ConfigTestSuite) TestNonExistingFile() {
	newPath := filepath.Join(s.TestDir, "new.yml")
	defaultPath := filepath.Join(s.TestDir, "home", ".srcd", "config.yml")

	testCases := []struct {
		name string
		path string
		args []string
	}{
		{
			name: "with --config",
			path: newPath,
			args: []string{"--config", newPath},
		},
		{
			name: "with defaults",
			path: defaultPath,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			r := s.RunCommand("config", tc.args...)
			require.NoError(r.Error, r.Combined())

			require.Equal(config.DefaultFileContents, r.Stdout())

			contents, err := ioutil.ReadFile(tc.path)
			require.NoError(err)
			require.Equal(config.DefaultFileContents, string(contents))
		})
	}
}

func (s *ConfigTestSuite) TestEmptyFile() {
	emptyPath := filepath.Join(s.TestDir, "empty.yml")
	defaultPath := filepath.Join(s.TestDir, "home", ".srcd", "config.yml")

	dir := filepath.Dir(defaultPath)
	err := os.MkdirAll(dir, 0755)
	s.Require().NoError(err)

	testCases := []struct {
		name string
		path string
		args []string
	}{
		{
			name: "with --config",
			path: emptyPath,
			args: []string{"--config", emptyPath},
		},
		{
			name: "with defaults",
			path: defaultPath,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			err := ioutil.WriteFile(tc.path, []byte("\n"), 0644)
			require.NoError(err)

			r := s.RunCommand("config", tc.args...)
			require.NoError(r.Error, r.Combined())

			require.Equal(config.DefaultFileContents, r.Stdout())

			contents, err := ioutil.ReadFile(tc.path)
			require.NoError(err)
			require.Equal(config.DefaultFileContents, string(contents))
		})
	}
}

func (s *ConfigTestSuite) TestExistingFile() {
	existingPath := filepath.Join(s.TestDir, "filled.yml")
	defaultPath := filepath.Join(s.TestDir, "home", ".srcd", "config.yml")

	dir := filepath.Dir(defaultPath)
	err := os.MkdirAll(dir, 0755)
	s.Require().NoError(err)

	testCases := []struct {
		name string
		path string
		args []string
	}{
		{
			name: "with --config",
			path: existingPath,
			args: []string{"--config", existingPath},
		},
		{
			name: "with defaults",
			path: defaultPath,
		},
	}

	st := `components:
  bblfshd:
    port: 1234`

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			err := ioutil.WriteFile(tc.path, []byte(st), 0644)
			require.NoError(err)

			r := s.RunCommand("config", tc.args...)
			require.NoError(r.Error, r.Combined())

			require.Equal(st, r.Stdout())

			contents, err := ioutil.ReadFile(tc.path)
			require.NoError(err)
			require.Equal(st, string(contents))
		})
	}
}
