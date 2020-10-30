package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/proxy"
)

func main() {
	cfg := proxy.NewConfig()
	if err := cfg.Parse(os.Args[1:]); err != nil {
		fmt.Printf("invalid command line argument, %v\n", err)
		os.Exit(1)
	}

	// setup shutdown handler.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// register shutdown hook.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case sig := <-sc:
			logger.Infof("signal %v received, waiting for multiplex to exit.", sig)
			cancel()
			logger.Infof("exiting...")
			logger.Unset()
			return
		}
	}()

	// run multiplex.
	m := proxy.NewMultiplex(cfg)
	if err := m.Run(ctx); err != nil {
		logger.Fatal(err)
	}
}
