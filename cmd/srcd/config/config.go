package config

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/src-d/engine/api"
	"gopkg.in/yaml.v2"
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
	} else {
		logrus.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}

	// The config file may define an int field as string, or have extra fields.
	// Calling Config we force a check on initialization.
	Config()
}

// Config returns the config used (from a file, env, or defaults)
func Config() *api.Config {
	var conf api.Config
	err := yaml.UnmarshalStrict([]byte(YamlStringConfig()), &conf)
	if err != nil {
		logrus.Fatalf("Config file does not follow the expected format: %s", err)
	}

	return &conf
}

// YamlStringConfig returns the CLI config used (from a file, env, or defaults)
// as a YAML string
func YamlStringConfig() string {
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		logrus.Fatalf("Unable to marshal config to YAML: %v", err)
	}
	return string(bs)
}
