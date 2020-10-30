package proxy

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dantin/logger"
)

const version = "0.0.1-dev"

type mirrorItem struct {
	ipAddr string
	port   int
}

// mirrorList represents comma separated mirror items.
type mirrorList []mirrorItem

func (l *mirrorList) String() string {
	return fmt.Sprint(*l)
}

// Set confirm flag Set interface.
func (l *mirrorList) Set(value string) error {
	for _, m := range strings.Split(value, ",") {
		tokens := strings.Split(m, ":")
		if len(tokens) != 2 {
			logger.Warnf("bad format of mirror item %s", m)
		}
		port, err := strconv.Atoi(tokens[1])
		if err != nil {
			logger.Warnf("bad port number of mirror item %s, caused by: %s", m, err)
			continue
		}

		*l = append(*l, mirrorItem{
			ipAddr: tokens[0],
			port:   port,
		})
	}
	return nil
}

// Config holds configuration of proxy.
type Config struct {
	*flag.FlagSet

	ListenAddr     *net.UDPAddr
	MirrorAddrs    mirrorList
	ConnectTimeout time.Duration
	ResolveTTL     time.Duration
}

// NewConfig creates an instance of UDP mutiplex configuration.
func NewConfig() *Config {
	return &Config{}
}

// Parse parses configuration from command line arguments.
func (cfg *Config) Parse(args []string) error {
	var (
		addr        string
		level       string
		showVersion bool
		showUsage   bool
	)
	appName := "udp-multiplex"

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.BoolVar(&showUsage, "h", false, "Show help message.")
	fs.StringVar(&addr, "l", "", "Listening address (e.g. 'localhost:8080').")
	fs.Var(&cfg.MirrorAddrs, "m", "Comma separated list of mirror addresses (e.g. 'localhost:8081,localhost:8082').")
	fs.DurationVar(&cfg.ConnectTimeout, "t", 500*time.Millisecond, "Client connect timeout")
	fs.DurationVar(&cfg.ResolveTTL, "ttl", 20*time.Millisecond, "Mirror resolve TTL")
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
	cfg.ListenAddr = serverAddr

	l, err := logger.New(level, os.Stdout)
	if err != nil {
		return fmt.Errorf("fail to resovle bind address, %v", err)
	}
	logger.Set(l)

	if cfg.ConnectTimeout.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of client connection timeout")
	}
	if cfg.ResolveTTL.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of mirror resolve TTL")
	}

	if len(cfg.MirrorAddrs) == 0 {
		return fmt.Errorf("mirror addresses are empty")
	}

	return nil
}
