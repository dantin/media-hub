package main

import (
	"log"
	"os"

	"github.com/dantin/media-hub/router"
)

const name = "srt-multiplex"

func main() {
	cfg := router.NewConfig(name)
	if err := cfg.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	svr := router.NewServer(cfg)
	if err := svr.Run(); err != nil {
		log.Fatal(err)
	}
}
