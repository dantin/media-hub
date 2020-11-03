package asset

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/dantin/logger"
)

func listenAndServe(addr string, mux *http.ServeMux, stop <-chan bool) error {
	shuttingDown := false

	httpdone := make(chan bool)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		var err error
		listenOn, err := net.Listen("tcp", addr)
		if err == nil {
			err = server.Serve(listenOn)
		}

		if err != nil {
			if shuttingDown {
				logger.Infof("HTTP server: stopped")
			} else {
				logger.Infof("HTTP server: failed", err)
			}
		}
		httpdone <- true
	}()

	// wait for either a termination signal or an error.
Loop:
	for {
		select {
		case <-stop:
			// flip the flat that we are terminating and close the Accept-ing socket, so no new connections are possible.
			shuttingDown = true
			// give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := server.Shutdown(ctx); err != nil {
				logger.Warnf("HTTP server failed to terminate gracefully, %v", err)
			}

			// wait for http server to stop Accept-ing connections.
			<-httpdone
			cancel()

			// Stop publishing statistics.
			statsShutdown()
			break Loop

		case <-httpdone:
			break Loop
		}
	}

	return nil
}
