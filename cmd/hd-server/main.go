package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dantin/logger"
)

func main() {
	l, err := logger.New("debug", os.Stdout)
	if err != nil {
		panic(err)
	}
	logger.Set(l)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sc
		fmt.Printf("%s - Shutdown signal received...\n", sig)
	}()

	logger.Debugf("haha")
}
