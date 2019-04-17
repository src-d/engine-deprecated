package cmd

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-log.v1"
)

type DeferedTestSuite struct {
	suite.Suite

	mockLogger *mockLogger
}

func TestDeferedTestSuite(t *testing.T) {
	suite.Run(t, new(DeferedTestSuite))
}

func (s *DeferedTestSuite) logAction(d *defered, timeout time.Duration) {
	s.T().Helper()

	s.mockLogger.Infof("Start")
	cancel := d.Print()
	time.Sleep(timeout)
	cancel()
	s.mockLogger.Infof("End")
}

func (s *DeferedTestSuite) buildDefered(withSpinner bool, inputFn func(stop <-chan bool) <-chan string) *defered {
	s.T().Helper()

	s.mockLogger = &mockLogger{}

	if withSpinner {
		d := newDefered(
			250*time.Millisecond,
			"Hello World!",
			nil,
			true,
			100*time.Millisecond,
		)
		d.logger = s.mockLogger
		d.logWriter = ioutil.Discard
		d.isTerminal = true

		return d
	}

	if inputFn != nil {
		d := newDefered(
			250*time.Millisecond,
			"Hello World!",
			inputFn,
			false,
			0,
		)
		d.logger = s.mockLogger
		d.logWriter = ioutil.Discard
		d.isTerminal = true

		return d
	}

	d := newDefered(
		250*time.Millisecond,
		"Hello World!",
		nil,
		false,
		0,
	)
	d.logger = s.mockLogger
	return d
}

func (s *DeferedTestSuite) TestPrint() {
	s.T().Run("timeout exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(false, nil)

		s.logAction(d, 500*time.Millisecond)

		expected := []string{"Start", "Hello World!", "End"}
		require.Equal(expected, s.mockLogger.msgs)
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(false, nil)

		s.logAction(d, 100*time.Millisecond)

		expected := []string{"Start", "End"}
		require.Equal(expected, s.mockLogger.msgs)
	})
}

func (s *DeferedTestSuite) TestPrintWithSpinner() {
	log.DefaultFactory = &log.LoggerFactory{
		Level:       log.InfoLevel,
		Format:      log.TextFormat,
		ForceFormat: true,
	}

	s.T().Run("timeout exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(true, nil)

		s.logAction(d, 500*time.Millisecond)

		expected := []string{
			"Start",
			"Hello World! ⠋",
			"Hello World! ⠙",
			"Hello World! ⠹",
			"Hello World!, done",
			"End",
		}
		require.Equal(expected, s.mockLogger.msgs)
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(true, nil)

		s.logAction(d, 100*time.Millisecond)

		expected := []string{"Start", "End"}
		require.Equal(expected, s.mockLogger.msgs)
	})
}

func (s *DeferedTestSuite) TestPrintWithInputFn() {
	inputFn := func(stop <-chan bool) <-chan string {
		ch := make(chan string)
		go func() {
			for {
				select {
				case <-stop:
					close(ch)
					return
				case <-time.After(100 * time.Millisecond):
					ch <- "Ping"
				}
			}
		}()
		return ch
	}

	s.T().Run("timeout exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(false, inputFn)

		s.logAction(d, 500*time.Millisecond)

		expected := []string{
			"Start",
			"Hello World!",
			"Ping",
			"Ping",
			"End",
		}
		require.Equal(expected, s.mockLogger.msgs)
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		d := s.buildDefered(false, inputFn)

		s.logAction(d, 100*time.Millisecond)

		expected := []string{"Start", "End"}
		require.Equal(expected, s.mockLogger.msgs)
	})
}

type mockLogger struct {
	msgs []string
}

func (l *mockLogger) New(log.Fields) log.Logger {
	return nil
}

func (l *mockLogger) With(log.Fields) log.Logger {
	return nil
}

func (l *mockLogger) Debugf(format string, args ...interface{}) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, args...))
}
func (l *mockLogger) Infof(format string, args ...interface{}) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, args...))

}
func (l *mockLogger) Warningf(format string, args ...interface{}) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, args...))
}
func (l *mockLogger) Errorf(err error, format string, args ...interface{}) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, args...))
}
