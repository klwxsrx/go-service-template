package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func TermSignalAwaiter(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-TermSignals():
	}

	return nil
}

func TermSignals() <-chan os.Signal { // TODO: wrap signals to ctx with signal.NotifyContext? test how worker.processes react to context.Cancelled
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	return ch
}
