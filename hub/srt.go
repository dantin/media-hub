package hub

import (
	"context"
	"time"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/subprocess"
)

//
func listenAndServe(listenAddr string, stop <-chan bool) error {
	errCh := make(chan error)

	server := subprocess.NewSubprocess(errCh, "/home/david/Documents/code/srt-live-server/bin/sls", nil, "-c", "/home/david/Documents/code/srt-live-server/sls.conf")

	if err := server.Run(); err != nil {
		logger.Warnf("SRT live server error, %v", err)
	}

	// wait for either a termination signal or an underlying error happens.
Loop:
	for {
		select {
		case <-stop:
			// give server 2 seconds to shut down.
			_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := server.Stop(); err != nil {
				logger.Warnf("SRT live server failed to terminate gracefully, %v", err)
			}
			cancel()

			break Loop
		case err := <-errCh:
			logger.Warnf("SRT live server error, %v", err)
		}
	}

	return nil
}
