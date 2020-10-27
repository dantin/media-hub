package main

import (
	"log"
	"os"

	"github.com/dantin/media-hub/proxy"
)

const name = "srt-multiplex"

func main() {
	cfg := proxy.NewConfig(name)
	if err := cfg.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	svr := proxy.NewServer(cfg)
	if err := svr.Run(); err != nil {
		log.Fatal(err)
	}
}
