package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/src-d/go-log.v1"
)

// ContextCommander is a cancellable commander. By default, receiving SIGTERM or
// SIGINT will cancel the context. For overriding the default behaviour, see the
// SignalHandler interface.
type ContextCommander interface {
	// ExecuteContext executes the command with a context.
	ExecuteContext(context.Context, []string) error
}

// SignalHandler can be implemented by a ContextCommander to override default
// signal handling (which is logging the signal and cancelling the context).
// If this interface is implemented, HandleSignal will be called when SIGTERM or
// SIGINT is received, and no other signal handling will be performed.
type SignalHandler interface {
	// HandleSignal takes the received signal (SIGTERM or SIGINT) and the
	// CancelFunc of the context.Context passed to ExecuteContext.
	HandleSignal(os.Signal, context.CancelFunc)
}

func executeContextCommander(cmd ContextCommander, args []string) error {
	handler := defaultSignalHandler
	if v, ok := cmd.(SignalHandler); ok {
		handler = v.HandleSignal
	}

	ctx := setupContext(handler)
	return cmd.ExecuteContext(ctx, args)
}

func defaultSignalHandler(signal os.Signal, cancel context.CancelFunc) {
	switch signal {
	case syscall.SIGTERM:
		log.Infof("signal SIGTERM received, stopping...")
	case os.Interrupt:
		log.Infof("signal SIGINT received, stopping...")
	}

	cancel()
}

func setupContext(handler func(os.Signal, context.CancelFunc)) context.Context {
	var (
		sigterm = make(chan os.Signal)
		sigint  = make(chan os.Signal)
	)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case sig := <-sigterm:
			handler(sig, cancel)
		case sig := <-sigint:
			handler(sig, cancel)
		}

		signal.Stop(sigterm)
		signal.Stop(sigint)
	}()

	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigint, os.Interrupt)

	return ctx
}
