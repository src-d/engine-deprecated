package api

import "github.com/sirupsen/logrus"

// NewLogEntry creates new LogEntry using logrus level and message
func NewLogEntry(level logrus.Level, msg string) *LogEntry {
	return &LogEntry{
		Level:   logrusToAPILevel[level],
		Message: msg,
	}
}

var logrusToAPILevel = map[logrus.Level]LogEntry_Level{
	logrus.InfoLevel:  LogEntry_INFO,
	logrus.WarnLevel:  LogEntry_WARN,
	logrus.ErrorLevel: LogEntry_ERROR,
}

// Print prints LogEntry to output using logrus
func (l *LogEntry) Print() {
	var log = logrus.Info
	switch l.Level {
	case LogEntry_WARN:
		log = logrus.Warn
	case LogEntry_ERROR:
		log = logrus.Error
	}

	log(l.Message)
}
