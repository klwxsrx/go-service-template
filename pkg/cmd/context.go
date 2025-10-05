package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// TODO: fix the context.Cancelled error in job loggers to warning level
func InitializeContext(signals ...os.Signal) (context.Context, context.CancelFunc) {
	defaultSignals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	if len(signals) == 0 {
		signals = defaultSignals
	}

	return signal.NotifyContext(context.Background(), signals...)
}
