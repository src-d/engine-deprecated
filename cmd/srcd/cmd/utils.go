package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Defered prints message only after specified timeout.
// Along with the message it supports a spinner or additional logs.
// These logs could be provided using the InputFn attribute.
type Defered struct {
	// Timeout is the amount of time to wait before showing the message
	Timeout time.Duration
	// Msg is the message to show after timeout
	Msg string
	// InputFn provides messages to display until cancel function is called
	// user MUST close channel when it finish
	InputFn func(stop <-chan bool) <-chan string
	// Spinner when set true shows spinner after the message
	Spinner bool
	// SpinnerInterval allows to change speed of spinner (200ms by default)
	SpinnerInterval time.Duration
}

// Print function would print message after timeout
// returns cancel function that should be called to cancel printing
func (d *Defered) Print() func() {
	ch := make(chan bool)
	done := make(chan bool)

	go func() {
		select {
		case <-time.After(d.Timeout):
			d.print(ch, done)
		case <-ch:
			done <- true
			return
		}
	}()

	return func() {
		close(ch)
		<-done
	}
}

func (d *Defered) print(ch <-chan bool, done chan<- bool) {
	// spinner doesn't support logs for now
	if d.Spinner {
		go d.spinner(d.Msg, ch, done)
		return
	}

	logrus.Info(d.Msg)
	if d.InputFn == nil {
		done <- true
		return
	}

	go func() {
		for line := range d.InputFn(ch) {
			logrus.Info(line)
		}

		done <- true
	}()
}

func (d *Defered) spinner(msg string, stop <-chan bool, done chan<- bool) {
	interval := d.SpinnerInterval
	if interval == 0 {
		interval = 200 * time.Millisecond
	}

	writer := logrus.StandardLogger().Out
	charset := []int{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	i := 0
	for {
		select {
		case <-stop:
			logrus.Infof("%s, done", d.Msg)
			done <- true
			return
		default:
			spinner := string(charset[i%len(charset)])
			logrus.Infof("%s %s", d.Msg, spinner)
			time.Sleep(interval)
			if writer == os.Stdout || writer == os.Stderr {
				fmt.Fprint(writer, "\033[A")
			}
		}

		i++
		if len(charset) == i {
			i = 0
		}
	}
}
