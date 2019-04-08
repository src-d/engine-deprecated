package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/src-d/engine/api"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-log.v1"
	yaml "gopkg.in/yaml.v2"
)

var configFile string

// InitConfig reads the config file if set.
func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		// Use config file from the flag.
		configFile = cfgFile
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return errors.Wrapf(err, "Could not detect home directory")
		}

		configFile = filepath.Join(home, ".srcd", "config.yml")
	}

	if configFile == "" {
		return nil
	}

	log.Debugf("Using config file: %s", configFile)

	// The config file may define an int field as string, or have extra fields.
	// Calling Config we force a check on initialization.
	_, err := Config()
	if err != nil {
		return errors.Wrapf(err, "Error checking config file '%s'", configFile)
	}

	return nil
}

// Config returns the config used
func Config() (*api.Config, error) {
	var conf api.Config

	if configFile == "" {
		return &conf, nil
	}

	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read config file %s", configFile)
	}

	err = yaml.UnmarshalStrict(content, &conf)
	if err != nil {
		return nil, errors.Wrap(err, "Config file does not follow the expected format")
	}

	return &conf, nil
}
