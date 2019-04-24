package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/src-d/engine/api"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

// DefaultFileContents is the default text for an empty config.yml file
var DefaultFileContents = `# Any change in the exposed ports will require you to run srcd init (or stop)

components:
  bblfshd:
    port: 9432

  bblfsh_web:
    port: 8081

  gitbase_web:
    port: 8080

  gitbase:
    port: 3306

  daemon:
    port: 4242
`

func init() {
	if runtime.GOOS == "windows" {
		DefaultFileContents = strings.Replace(DefaultFileContents, "\n", "\r\n", -1)
	}
}

// File contains the config read from the file path used in Read
var File = &api.Config{}

// Read reads the config file values into File. If configFile path is empty,
// $HOME/.srcd/config.yml will be used, only if it exists.
// If configFile is empty and the default file does not exist the return value
// is nil
func Read(configFile string) error {
	if configFile == "" {
		var err error
		if configFile, err = DefaultPath(); err != nil {
			return err
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return nil
		}
	}

	log.Debugf("Using config file: %s", configFile)

	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read config file %s", configFile)
	}

	err = yaml.UnmarshalStrict(content, File)
	if err != nil {
		return errors.Wrapf(err, "config file %s does not follow the expected format", configFile)
	}

	return nil
}

// DefaultPath returns the default config file path, $HOME/.srcd/config.yml
func DefaultPath() (string, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "could not detect home directory")
	}

	return filepath.Join(homedir, ".srcd", "config.yml"), nil
}
