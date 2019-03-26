package api

import (
	"github.com/src-d/engine/components"
)

// Config holds the config.yml file values
type Config struct {
	Components struct {
		Bblfshd struct {
			// Port is the public exposed port for this component's container
			Port int
		}

		BblfshWeb struct {
			// Port is the public exposed port for this component's container
			Port int
		} `yaml:"bblfsh_web"`

		GitbaseWeb struct {
			// Port is the public exposed port for this component's container
			Port int
		} `yaml:"gitbase_web"`

		Gitbase struct {
			// Port is the public exposed port for this component's container
			Port int
		}

		Daemon struct {
			// Port is the public exposed port for the daemon container
			Port int
		}
	}
}

// SetDefaults fills the default values for any fields that are not set
func (c *Config) SetDefaults() {
	if c.Components.Bblfshd.Port == 0 {
		c.Components.Bblfshd.Port = components.BblfshParsePort
	}

	if c.Components.BblfshWeb.Port == 0 {
		c.Components.BblfshWeb.Port = 8081
	}

	if c.Components.GitbaseWeb.Port == 0 {
		c.Components.GitbaseWeb.Port = 8080
	}

	if c.Components.Gitbase.Port == 0 {
		c.Components.Gitbase.Port = components.GitbasePort
	}

	if c.Components.Daemon.Port == 0 {
		c.Components.Daemon.Port = components.DaemonPort
	}
}
