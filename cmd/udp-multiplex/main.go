package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/proxy"
)

const (
	appName = "udp-multiplex"
	version = "0.0.1-dev"

	maxBufferSize = 10 * (1 << 10) // 10k bit buffer size.
)

var (
	listenAddr     *net.UDPAddr
	mirrorAddrs    proxy.MirrorList
	connectTimeout time.Duration
	resolveTTL     time.Duration

	showVersion bool
	showUsage   bool
)

func parseArgs(args []string) error {
	var (
		addr  string
		level string
	)

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.BoolVar(&showUsage, "h", false, "Show help message.")
	fs.StringVar(&addr, "l", "", "Listening address (e.g. 'localhost:8080').")
	fs.Var(&mirrorAddrs, "m", "Comma separated list of mirror addresses (e.g. 'localhost:8081,localhost:8082').")
	fs.DurationVar(&connectTimeout, "t", 500*time.Millisecond, "Client connect timeout")
	fs.DurationVar(&resolveTTL, "ttl", 20*time.Millisecond, "Mirror resolve TTL")
	fs.StringVar(&level, "level", "info", "Log level, supported level: debug, info, error, fatal.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) > 0 {
		return fmt.Errorf("%q is not a valid flag", fs.Arg(0))
	}

	if showVersion {
		fmt.Printf("%s %s\n", appName, version)
		os.Exit(0)
	}

	if showUsage {
		fs.Usage()
		os.Exit(0)
	}
	if addr == "" {
		return fmt.Errorf("listen address is empty")
	}

	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("fail to resovle bind address, %v", err)
	}
	listenAddr = serverAddr

	l, err := logger.New(level, os.Stdout)
	if err != nil {
		return fmt.Errorf("fail to resovle bind address, %v", err)
	}
	logger.Set(l)

	if connectTimeout.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of client connection timeout")
	}
	if resolveTTL.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of mirror resolve TTL")
	}

	if len(mirrorAddrs) == 0 {
		return fmt.Errorf("mirror addresses are empty")
	}

	return nil
}

func main() {
	if err := parseArgs(os.Args[1:]); err != nil {
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
			return
		}
	}()

	// run multiplex.
	m := proxy.NewMultiplex(listenAddr, mirrorAddrs, connectTimeout, resolveTTL, maxBufferSize)
	if err := m.Run(ctx); err != nil {
		logger.Fatal(err)
	}
}
