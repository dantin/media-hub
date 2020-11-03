package main

import (
	"fmt"
	"os"

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

	svr := asset.NewServer(cfg)
	if err := svr.Run(); err != nil {
		logger.Fatal(err)
	}
}
