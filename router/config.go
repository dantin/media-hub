package router

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const version = "0.0.1-dev"

var (
	writeTimeout   time.Duration
	connectTimeout time.Duration
)

type mirrorList []*net.UDPAddr

func (l *mirrorList) String() string {
	return fmt.Sprint(*l)
}

func (l *mirrorList) Set(value string) error {
	for _, m := range strings.Split(value, ",") {
		addr, err := net.ResolveUDPAddr("udp", m)
		if err != nil {
			log.Printf("ignore bad udp address %s, caused by: %s", m, err)
			continue
		}

		*l = append(*l, addr)
	}
	return nil
}

// Config represents the configuration.
type Config struct {
	*flag.FlagSet

	ListenAddr  string
	MirrorAddrs mirrorList

	showVersion bool
	showUsage   bool
}

// NewConfig returns an instance of configuration.
func NewConfig(name string) *Config {
	cfg := Config{}
	cfg.FlagSet = flag.NewFlagSet(name, flag.ContinueOnError)
	fs := cfg.FlagSet
	fs.BoolVar(&cfg.showVersion, "v", false, "Print version information.")
	fs.BoolVar(&cfg.showUsage, "h", false, "Show help message.")
	fs.StringVar(&cfg.ListenAddr, "l", "", "Listening address (e.g. 'localhost:8080').")
	fs.Var(&cfg.MirrorAddrs, "m", "Comma separated list of mirror addresses (e.g. 'localhost:8081,localhost:8082').")
	fs.DurationVar(&connectTimeout, "t", 500*time.Millisecond, "Mirror connect timeout")
	fs.DurationVar(&writeTimeout, "d", 20*time.Millisecond, "Mirror write timeout")

	return &cfg
}

// Parse checks a command line arguments array.
func (cfg *Config) Parse(args []string) error {
	_ = cfg.FlagSet.Parse(args)
	if len(cfg.FlagSet.Args()) > 0 {
		return fmt.Errorf("%q is not a valid flag", cfg.FlagSet.Arg(0))
	}

	if cfg.showVersion {
		log.Printf(version)
		os.Exit(0)
	}

	if cfg.showUsage {
		cfg.FlagSet.Usage()
		os.Exit(0)
	}

	if cfg.ListenAddr == "" || len(cfg.MirrorAddrs) == 0 {
		return fmt.Errorf("listen address or mirror addresses are empty")
	}

	return nil
}
