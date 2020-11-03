package main

import (
	"fmt"
	"os"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/proxy"
)

func main() {
	defer logger.Unset()

	cfg := proxy.NewConfig()
	if err := cfg.Parse(os.Args[1:]); err != nil {
		fmt.Printf("configuration parsing error, %v\n", err)
		os.Exit(1)
	}

	// run multiplex.
	m := proxy.NewMultiplex(cfg)
	if err := m.Run(); err != nil {
		logger.Fatal(err)
	}
}
