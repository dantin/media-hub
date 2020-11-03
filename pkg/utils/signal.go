package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/dantin/logger"
)

// SignalHandler returns a channel that emit a message when a signal happens.
func SignalHandler() <-chan bool {
	stop := make(chan bool)

	// setup shutdown handler.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		// Wait for a signal. Don't care with signal it is
		sig := <-sc
		logger.Infof("Signal %v received, shutting down", sig)
		stop <- true
	}()

	return stop
}
