package cli

import (
	"gopkg.in/src-d/go-log.v1"
)

// LogOptions defines logging flags. It is meant to be embedded in a
// command struct.
type LogOptions struct {
	LogLevel       string `long:"log-level" env:"LOG_LEVEL" choice:"info" choice:"debug" choice:"warning" choice:"error" default:"info" description:"Logging level"`
	LogFormat      string `long:"log-format" env:"LOG_FORMAT" choice:"text" choice:"json" description:"log format, defaults to text on a terminal and json otherwise"`
	LogFields      string `long:"log-fields" env:"LOG_FIELDS" description:"default fields for the logger, specified in json"`
	LogForceFormat bool   `long:"log-force-format" env:"LOG_FORCE_FORMAT" description:"ignore if it is running on a terminal or not"`
}

// Init initializes the default logger factory.
func (c LogOptions) Init(a *App) error {
	log.DefaultFactory = &log.LoggerFactory{
		Level:       c.LogLevel,
		Format:      c.LogFormat,
		Fields:      c.LogFields,
		ForceFormat: c.LogForceFormat,
	}

	log.DefaultLogger = log.New(nil)
	return nil
}
