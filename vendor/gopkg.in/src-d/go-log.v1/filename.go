package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// maxStackFrames stands for maximum number of stack frames.
// This const is used as a maximum number of stack frames to skip.
// It prevents from going to deep on stack (due to performance reasons).
const maxStackFrames = 16

const unknownFilename = "???"

// filenameHook implements logrus.Hook interface
type filenameHook struct {
	field      string   // field is a name used in logging
	skipframes int      // skipframes is used to skip number of stack frames.
	skipnames  []string // skipnames contains stack frames prefixes to skip.
	levels     []logrus.Level
	formatter  func(file string, line int) string
}

func (hook *filenameHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *filenameHook) Fire(entry *logrus.Entry) error {
	file, line := hook.caller()
	entry.Data[hook.field] = hook.formatter(file, line)
	return nil
}

// newFilenameHook creates a new default filename hook.
// By default:
// -  skips 5 stack frames,
// - ignore internal calls for logrus and go-log
// - logging in following format "source={file}:{line}"
func newFilenameHook(levels ...logrus.Level) *filenameHook {
	hook := filenameHook{
		field:      "source",
		skipframes: 5,
		skipnames:  []string{"go-log", "logrus"},
		levels:     levels,
		formatter: func(file string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		},
	}
	if len(hook.levels) == 0 {
		hook.levels = logrus.AllLevels
	}

	return &hook
}

// caller returns filename with base directory (e.g.: "go-log.v1/logger.go")
// and line number
func (hook *filenameHook) caller() (string, int) {
	for i := hook.skipframes; i < maxStackFrames; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		file = basename(file)
		if !hasPrefix(file, hook.skipnames...) {
			return file, line
		}
	}

	return unknownFilename, 0
}

// basename returns file name and base directory e.g.:
// for path "gopkg.in/src-d/go-log.v1/logger.go"
// function returns "go-log.v1/logger.go"
func basename(path string) string {
	i, n := len(path)-1, 2
	vollen := len(filepath.VolumeName(path))

	for i >= vollen {
		if os.IsPathSeparator(path[i]) {
			n--
			if n == 0 {
				break
			}
		}

		i--
	}

	return path[i+1:]
}

func hasPrefix(s string, prefix ...string) bool {
	for _, p := range prefix {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
