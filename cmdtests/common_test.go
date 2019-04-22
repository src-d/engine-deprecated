// +build integration

package cmdtests_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/src-d/engine/cmdtests"
	"github.com/stretchr/testify/suite"
)

type StreamLinifierTestSuite struct {
	suite.Suite
}

func TestStreamLinifierTestSuite(t *testing.T) {
	suite.Run(t, &StreamLinifierTestSuite{})
}

func (s *StreamLinifierTestSuite) TestOneLineOnOneMessage() {
	require := s.Require()

	sl := cmdtests.NewStreamLinifier(500 * time.Millisecond)

	in := make(chan string)
	messagesNum := 3

	go func() {
		for i := 1; i <= messagesNum; i++ {
			in <- fmt.Sprintf("line %d\n", i)
		}

		close(in)
	}()

	out := sl.Linify(in)
	counter := 0
	for m := range out {
		counter++
		require.Equal(m, fmt.Sprintf("line %d", counter))
	}

	require.Equal(messagesNum, counter)
}

func (s *StreamLinifierTestSuite) TestOneLineOnMultiMessages() {
	require := s.Require()

	sl := cmdtests.NewStreamLinifier(1 * time.Second)

	in := make(chan string)

	go func() {
		in <- "line 1 (1/3)|"
		in <- "line 1 (2/3)|"
		in <- "line 1 (3/3)\n"

		in <- "line 2 (1/2)|"
		in <- "line 2 (2/2)\n"

		in <- "line 3 (1/1)\n"

		close(in)
	}()

	out := sl.Linify(in)
	require.Equal(<-out, "line 1 (1/3)|line 1 (2/3)|line 1 (3/3)")
	require.Equal(<-out, "line 2 (1/2)|line 2 (2/2)")
	require.Equal(<-out, "line 3 (1/1)")
}

func (s *StreamLinifierTestSuite) TestMultiLinesOnOneMessage() {
	require := s.Require()

	sl := cmdtests.NewStreamLinifier(1 * time.Second)

	in := make(chan string)

	go func() {
		in <- "line 1\nline 2\nline 3\n"

		close(in)
	}()

	out := sl.Linify(in)
	require.Equal(<-out, "line 1")
	require.Equal(<-out, "line 2")
	require.Equal(<-out, "line 3")
}

func (s *StreamLinifierTestSuite) TestSlowPendingMessage() {
	require := s.Require()

	sl := cmdtests.NewStreamLinifier(1 * time.Second)

	in := make(chan string)

	go func() {
		in <- "line 1 (1/3)|"
		time.Sleep(1 * time.Second)
		in <- "line 1 (2/3)|"
		time.Sleep(1 * time.Second)
		in <- "line 1 (3/3)\n"

		close(in)
	}()

	out := sl.Linify(in)
	require.Equal(<-out, "line 1 (1/3)|")
	require.Equal(<-out, "line 1 (2/3)|")
	require.Equal(<-out, "line 1 (3/3)")
}
