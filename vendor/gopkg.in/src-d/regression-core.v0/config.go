package regression

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Config holds the general configuration for tests
type Config struct {
	// Versions has the list of releases to test
	Versions []string
	// OS holds the operating system
	OS string
	// BinaryCache is the path to the binaries cache
	BinaryCache string `env:"REG_BINARIES" default:"binaries" long:"binaries" description:"Directory to store binaries"`
	// RepositoriesCache is the path to the downloaded repositories
	RepositoriesCache string `env:"REG_REPOS" default:"repos" long:"repos" description:"Directory to store repositories"`
	// GitURL is the git repository url to download the tool
	GitURL string `env:"REG_GITURL" default:"" long:"url" description:"URL to the tool repo"`
	// GitServerPort is the port where the local git server will listen
	GitServerPort int `env:"REG_GITPORT" default:"9418" long:"gitport" description:"Port for local git server"`
	// RepositoriesFile
	RepositoriesFile string `env:"REG_REPOS_FILE" default:"" long:"repos-file" description:"YAML file with the list of repos"`
	// Complexity has the max number of complexity of repos to test
	Complexity int `env:"REG_COMPLEXITY" default:"1" long:"complexity" short:"c" description:"Complexity of the repositories to test"`
	// Repeat is the number of times each test will be run
	Repeat int `env:"REG_REPEAT" default:"3" long:"repeat" short:"n" description:"Number of times a test is run"`
	// ShowRepos when --show-repos is specified
	ShowRepos bool `long:"show-repos" description:"List available repositories to test"`
	// GitHubToken specifies the token to use to use GitHub API
	GitHubToken string `env:"REG_TOKEN" long:"token" short:"t" description:"Token used to connect to the API"`
}

// NewConfig returns an empty config with initialized OS.
func NewConfig() Config {
	return Config{
		OS: runtime.GOOS,
	}
}

// VersionPath returns the path of the binary cache for an specific version.
func (c *Config) VersionPath(version string) string {
	return filepath.Join(c.BinaryCache, version)
}

// VersionPath returns the binary path an specific version.
func (c *Config) BinaryPath(version, name string) string {
	return filepath.Join(c.VersionPath(version), name)
}

// BuildStep describes a command used to build a tool.
type BuildStep struct {
	// Dir is the path where this command is executed.
	Dir string
	// Command is the executable to run.
	Command string
	// Args caintains the list of options to use with Command.
	Args []string
}

// Tool describes a project to build and test.
type Tool struct {
	// Name has the tool name.
	Name string
	// BinaryName has the executable name of the tool.
	BinaryName string
	// GitURL holds the git URL to download the project.
	GitURL string
	// ProjectPath is the directory structure inside GOPATH/src where it should
	// be located for building.
	ProjectPath string
	// BuildSteps has the commands needed to build the tool.
	BuildSteps []BuildStep
	// ExtraFiles is a list of files used from the repository.
	ExtraFiles []string
}

func (t Tool) DirName(os string) string {
	return fmt.Sprintf("%s_%s_amd64", t.Name, os)
}

func (t Tool) BinName() string {
	if t.BinaryName == "" {
		return t.Name
	}

	return t.BinaryName
}
