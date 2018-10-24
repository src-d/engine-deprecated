package engine

import (
	"context"

	"github.com/sirupsen/logrus"
)

// deprecated: used for parseWithLogs, better refactor using context
type logf func(format string, args ...interface{})

// logfn is a hook for logrus
type logfn func(level logrus.Level, msg string)

func (fn logfn) Levels() []logrus.Level {
	// we don't need to connect to fatal messages because they will be logged on client anyway
	// we may add debug level if we see need for it
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

func (fn logfn) Fire(e *logrus.Entry) error {
	fn(e.Level, e.Message)
	return nil
}

// unique type for context key
type logfCtxType int

const logfKey logfCtxType = 1

// getLogger returns logger with appended hook from context
func getLogger(ctx context.Context) logrus.FieldLogger {
	fn, ok := ctx.Value(logfKey).(logfn)
	if !ok {
		return logrus.StandardLogger()
	}

	logger := logrus.New()
	logger.AddHook(fn)

	return logger
}

// setLogf sets logger hook to context
func setLogf(ctx context.Context, fn logfn) context.Context {
	return context.WithValue(ctx, logfKey, fn)
}
