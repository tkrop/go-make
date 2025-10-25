// Package sys provides utilities for working with system signals.
package sys

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// Signals a common and extensive list of operating system signals to wait
// for before shutting down a service or daemon.
var Signals = []os.Signal{
	syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT,
	syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGSTOP,
}

// SignalFunc defines the function signature for signal handler functions.
type SignalFunc func(cancel context.CancelFunc, signal os.Signal)

// Signaler provides a signal handler that listens for configured signals
// and calls the provided signal handler function when a signal is received.
type Signaler struct {
	handler SignalFunc
	signals []os.Signal
	channel chan os.Signal
}

// NewSignaler returns a new signal handler that listens for the given
// signals and calls the provided signal handler function when a signal is
// received. To activate the signal handler, call the `Handle` method with a
// context to setup the cancel context and start listening for signals.
func NewSignaler(handler SignalFunc, signals ...os.Signal) *Signaler {
	return &Signaler{
		handler: handler,
		signals: signals,
		channel: make(chan os.Signal, 1),
	}
}

// Signal extends the given context with a cancel context and starts listening
// for the configured signals to call the provided signal handler function.
func (s *Signaler) Signal(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	signal.Notify(s.channel, s.signals...)
	go s.signal(ctx, cancel)

	return ctx
}

// signal waits for signals and calls the provided signal handler function
// when a signal is received or exits when the context is done.
func (s *Signaler) signal(
	ctx context.Context, cancel context.CancelFunc,
) {
	select {
	case signal := <-s.channel:
		s.handler(cancel, signal)
	case <-ctx.Done():
	}
}
