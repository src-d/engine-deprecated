package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/src-d/go-log.v1"
)

// defered prints message only after specified timeout.
// Along with the message it supports a spinner or additional logs.
// These logs could be provided using the InputFn attribute.
type defered struct {
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

	// logger is the go-log DefaultLogger. Can be changed for tests
	logger log.Logger
	// logWriter is the io.Writer where logger sends the output. By default,
	// os.Stderr. Can be changed for tests
	logWriter io.Writer
	// isTerminal can be set to force the spinner to be printed without checking
	// if the output is a terminal. For tests
	isTerminal bool
}

func newDefered(
	timeout time.Duration,
	msg string,
	inputFn func(stop <-chan bool) <-chan string,
	spinner bool,
	spinnerInterval time.Duration,
) *defered {
	return &defered{
		Timeout:         timeout,
		Msg:             msg,
		InputFn:         inputFn,
		Spinner:         spinner,
		SpinnerInterval: spinnerInterval,
		logger:          log.DefaultLogger,
		logWriter:       os.Stderr,
		isTerminal:      terminal.IsTerminal(int(os.Stderr.Fd())),
	}
}

// Print function would print message after timeout
// returns cancel function that should be called to cancel printing
func (d *defered) Print() func() {
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

func (d *defered) print(ch <-chan bool, done chan<- bool) {
	// spinner doesn't support logs for now
	if d.Spinner {
		go d.spinner(d.Msg, ch, done)
		return
	}

	d.logger.Infof(d.Msg)
	if d.InputFn == nil {
		done <- true
		return
	}

	go func() {
		for line := range d.InputFn(ch) {
			d.logger.Infof(line)
		}

		done <- true
	}()
}

func (d *defered) spinner(msg string, stop <-chan bool, done chan<- bool) {
	// skip if log level is not info or debug. We print directly into os.Stderr,
	// so we cannot rely on log.Infof skipping the message
	if log.DefaultFactory.Level != log.InfoLevel &&
		log.DefaultFactory.Level != log.DebugLevel {
		return
	}

	interval := d.SpinnerInterval
	if interval == 0 {
		interval = 200 * time.Millisecond
	}

	// If the logger format is not text, or the output is not a terminal,
	// do not print the spinner
	if log.DefaultFactory.Format != log.TextFormat || !d.isTerminal {
		d.logger.Infof(d.Msg)

		select {
		case <-stop:
			d.logger.Infof("%s, done", d.Msg)
			done <- true
			return
		}
	}

	charset := []int{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	i := 0
	for {
		select {
		case <-stop:
			d.logger.Infof("%s, done", d.Msg)
			done <- true
			return
		default:
			spinner := string(charset[i%len(charset)])
			d.logger.Infof("%s %s", d.Msg, spinner)
			time.Sleep(interval)
			fmt.Fprint(d.logWriter, "\033[A")
		}

		i++
		if len(charset) == i {
			i = 0
		}
	}
}
