package config

import (
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/src-d/engine/api"
	yaml "gopkg.in/yaml.v2"
)

// InitConfig reads the config file and ENV variables if set.
func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			logrus.Fatalf("Could not detect home directory: %s", err)
		}

		// Search config in $HOME/.srcd/ with name "config" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".srcd"))
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		// viper returns ConfigFileNotFoundError only when the default config file
		// is not found. For a wrong file path on --config it's an os.PathError
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logrus.Fatalf("Error reading the config file %s: %s", viper.ConfigFileUsed(), err.Error())
		}
	}

	checkConfig()
}

func checkConfig() {
	actualConfigFile := viper.ConfigFileUsed()
	if actualConfigFile == "" {
		return
	}

	logrus.Debugf("Using config file: %s", actualConfigFile)

	// The config file may define an int field as string, or have extra fields.
	// Calling Config we force a check on initialization.
	_, err := Config()
	if err != nil {
		logrus.Fatal(errors.Wrapf(err, "Error checking config file '%s'",
			actualConfigFile))
	}
}

// Config returns the config used (from a file, env, or defaults)
func Config() (*api.Config, error) {
	var conf api.Config
	err := yaml.UnmarshalStrict([]byte(yamlStringConfig()), &conf)
	if err != nil {
		return nil, errors.Wrap(err, "Config file does not follow the expected format")
	}

	return &conf, nil
}

// yamlStringConfig returns the CLI config used (from a file, env, or defaults)
// as a YAML string
func yamlStringConfig() string {
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		logrus.Fatalf("Unable to marshal config to YAML: %v", err)
	}
	return string(bs)
}
