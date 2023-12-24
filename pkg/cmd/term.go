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
	case <-TermSignals():
	}

	return nil
}

func TermSignals() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	return ch
}
