package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/asset"
)

func main() {
	defer logger.Unset()

	cfg := asset.NewConfig()
	if err := cfg.Parse(os.Args[1:]); err != nil {
		fmt.Printf("configuration parsing error, %v\n", err)
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
	_, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case sig := <-sc:
			logger.Infof("signal %v received, waiting to exit.", sig)
			cancel()
			logger.Infof("exiting...")
			return
		}
	}()

	// run server.
	svr := asset.NewServer(cfg)
	if err := svr.Run(); err != nil {
		logger.Fatal(err)
	}
}
