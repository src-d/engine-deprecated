// +build !integration

package cmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	cmdtest "github.com/src-d/engine/cmd/test-utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DeferedTestSuite struct {
	suite.Suite
}

func TestDeferedTestSuite(t *testing.T) {
	suite.Run(t, new(DeferedTestSuite))
}

func (s *DeferedTestSuite) buildFn(d *Defered, timeout time.Duration) func() {
	s.T().Helper()

	return func() {
		logrus.Info("Start")
		cancel := d.Print()
		time.Sleep(timeout)
		cancel()
		logrus.Info("End")
	}
}

func (s *DeferedTestSuite) buildDefered(withSpinner bool, inputFn func(stop <-chan bool) <-chan string) *Defered {
	s.T().Helper()

	if withSpinner {
		return &Defered{
			Timeout:         250 * time.Millisecond,
			Msg:             "Hello World!",
			Spinner:         true,
			SpinnerInterval: 100 * time.Millisecond,
		}
	}

	if inputFn != nil {
		return &Defered{
			Timeout: 250 * time.Millisecond,
			Msg:     "Hello World!",
			InputFn: inputFn,
		}
	}

	return &Defered{
		Timeout: 250 * time.Millisecond,
		Msg:     "Hello World!",
	}
}

func (s *DeferedTestSuite) TestPrint() {
	s.T().Run("timeout exceeded", func(t *testing.T) {
		require := require.New(t)

		var memLog bytes.Buffer
		d := s.buildDefered(false, nil)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 500*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 3)
		expected := [3]string{"Start", "Hello World!", "End"}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		var memLog bytes.Buffer
		d := s.buildDefered(false, nil)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 100*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 2)
		expected := [2]string{"Start", "End"}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
	})
}

func (s *DeferedTestSuite) TestPrintWithSpinner() {
	s.T().Run("timeout exceeded", func(t *testing.T) {
		require := require.New(t)

		var memLog bytes.Buffer
		d := s.buildDefered(true, nil)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 500*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 6)
		expected := [6]string{
			"Start",
			"Hello World! ⠋",
			"Hello World! ⠙",
			"Hello World! ⠹",
			"Hello World!, done",
			"End",
		}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		var memLog bytes.Buffer
		d := s.buildDefered(true, nil)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 100*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 2)
		expected := [2]string{"Start", "End"}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
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

		var memLog bytes.Buffer
		d := s.buildDefered(false, inputFn)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 500*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 5)
		expected := [5]string{
			"Start",
			"Hello World!",
			"Ping",
			"Ping",
			"End",
		}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
	})

	s.T().Run("timeout not exceeded", func(t *testing.T) {
		require := require.New(t)

		var memLog bytes.Buffer
		d := s.buildDefered(false, inputFn)

		logMessages := cmdtest.TraceLogMessages(s.buildFn(d, 100*time.Millisecond), &memLog)

		require.Equal(len(logMessages), 2)
		expected := [2]string{"Start", "End"}
		for i, lm := range logMessages {
			require.Equal(lm.Msg, expected[i])
		}
	})
}
