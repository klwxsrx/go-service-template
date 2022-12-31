package sig

import (
	"os"
	"os/signal"
	"syscall"
)

func Wait(termSignalsChan <-chan os.Signal) {
	<-termSignalsChan
}

func TermSignals() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	return ch
}
