package log

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	// DebugLevel stands for debug logging level.
	DebugLevel = "debug"
	// InfoLevel stands for info logging level (default).
	InfoLevel = "info"
	// WarningLevel stands for warning logging level.
	WarningLevel = "warning"
	// ErrorLevel stands for error logging level.
	ErrorLevel = "error"

	// disabled is a special level used only when we are in test
	disabledLevel = "panic"

	// TextFormat stands for text logging format.
	TextFormat = "text"
	// JSONFormat stands for json logging format.
	JSONFormat = "json"

	// DefaultLevel is the level used by LoggerFactory when Level is omitted.
	DefaultLevel = InfoLevel
	// DefaultFormat is the format used by LoggerFactory when Format is omitted.
	DefaultFormat = TextFormat
)

var (
	validLevels = map[string]bool{
		InfoLevel: true, DebugLevel: true, WarningLevel: true, ErrorLevel: true,
		disabledLevel: true,
	}
	validFormats = map[string]bool{
		TextFormat: true, JSONFormat: true,
	}
)

// LoggerFactory is a logger factory used to instanciate new Loggers, from
// string configuration, mainly coming from console flags.
type LoggerFactory struct {
	// Level as string, values are "info", "debug", "warning" or "error".
	Level string
	// Format as string, values are "text" or "json", by default "text" is used.
	// when a terminal is not detected "json" is used instead.
	Format string
	// Fields in JSON format to be used by configured in the new Logger.
	Fields string
	// ForceFormat if true the fact of being in a terminal or not is ignored.
	ForceFormat bool
}

// New returns a new logger based on the LoggerFactory values.
func (f *LoggerFactory) New(fields Fields) (Logger, error) {
	l := logrus.New()

	if err := f.setLevel(l); err != nil {
		return nil, err
	}

	f.setHook(l)

	if err := f.setFormat(l); err != nil {
		return nil, err
	}

	return f.setFields(l, fields)
}

// ApplyToLogrus configures the standard logrus Logger with the LoggerFactory
// values. Useful to propagate the configuration to third-party libraries using
// logrus.
func (f *LoggerFactory) ApplyToLogrus() error {
	std := logrus.StandardLogger()
	if err := f.setLevel(std); err != nil {
		return err
	}
	f.setHook(std)

	return f.setFormat(std)
}

func (f *LoggerFactory) setLevel(l *logrus.Logger) error {
	if err := f.setDefaultLevel(); err != nil {
		return err
	}

	level, err := logrus.ParseLevel(f.Level)
	if err != nil {
		return err
	}

	l.Level = level
	return nil
}

func (f *LoggerFactory) setDefaultLevel() error {
	if f.Level == "" {
		f.Level = DefaultLevel
	}

	f.Level = strings.ToLower(f.Level)
	if validLevels[f.Level] {
		return nil
	}

	return fmt.Errorf(
		"invalid level %s, valid levels are: %v",
		f.Level, getKeysFromMap(validLevels),
	)
}

func (f *LoggerFactory) setFormat(l *logrus.Logger) error {
	if err := f.setDefaultFormat(); err != nil {
		return err
	}

	switch f.Format {
	case "text":
		f := new(prefixed.TextFormatter)
		f.ForceColors = true
		f.FullTimestamp = true
		f.TimestampFormat = time.RFC3339Nano
		l.Formatter = f
	case "json":
		f := new(logrus.JSONFormatter)
		f.TimestampFormat = time.RFC3339Nano
		l.Formatter = f
	}

	return nil
}

func (f *LoggerFactory) setDefaultFormat() error {
	if f.Format == "" {
		f.Format = DefaultFormat
	}

	f.Format = strings.ToLower(f.Format)
	if validFormats[f.Format] {
		return nil
	}

	if !f.ForceFormat && isTerminal() {
		f.Format = JSONFormat
	}

	return fmt.Errorf(
		"invalid format %s, valid formats are: %v",
		f.Format, getKeysFromMap(validFormats),
	)
}

func (f *LoggerFactory) setHook(l *logrus.Logger) {
	if f.Level == DebugLevel {
		l.AddHook(newFilenameHook(
			logrus.DebugLevel,
			logrus.InfoLevel,
			logrus.WarnLevel,
			logrus.ErrorLevel,
			logrus.PanicLevel),
		)
	}
}

func (f *LoggerFactory) setFields(l *logrus.Logger, fields Fields) (Logger, error) {
	var envFields logrus.Fields
	if f.Fields != "" {
		if err := json.Unmarshal([]byte(f.Fields), &envFields); err != nil {
			return nil, err
		}
	}

	if envFields == nil {
		envFields = make(logrus.Fields, 0)
	}

	if fields != nil {
		for k, v := range fields {
			envFields[k] = v
		}
	}

	e := l.WithFields(envFields)
	return &logger{*e}, nil
}

func getKeysFromMap(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func isTerminal() bool {
	return terminal.IsTerminal(int(os.Stdout.Fd()))
}
