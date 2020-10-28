package proxy

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const version = "0.0.1-dev"

type mirrorItem struct {
	ipAddr string
	port   int
}
type mirrorList []mirrorItem

func (l *mirrorList) String() string {
	return fmt.Sprint(*l)
}

func (l *mirrorList) Set(value string) error {
	for _, m := range strings.Split(value, ",") {
		tokens := strings.Split(m, ":")
		if len(tokens) != 2 {
			log.Printf("bad format of mirror item %s", m)
		}
		port, err := strconv.Atoi(tokens[1])
		if err != nil {
			log.Printf("bad port number of mirror item %s, caused by: %s", m, err)
			continue
		}

		*l = append(*l, mirrorItem{
			ipAddr: tokens[0],
			port:   port,
		})
	}
	return nil
}

// Config represents the configuration.
type Config struct {
	*flag.FlagSet

	ListenAddr     string
	MirrorAddrs    mirrorList
	ConnectTimeout time.Duration
	ResolveTTL     time.Duration

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
	fs.DurationVar(&cfg.ConnectTimeout, "t", 500*time.Millisecond, "Client connect timeout")
	fs.DurationVar(&cfg.ResolveTTL, "d", 20*time.Millisecond, "Mirror resolve TTL")

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

	if cfg.ConnectTimeout.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of client connection timeout")
	}
	if cfg.ResolveTTL.Nanoseconds() == 0 {
		return fmt.Errorf("invalid value of mirror resolve TTL")
	}

	if cfg.ListenAddr == "" || len(cfg.MirrorAddrs) == 0 {
		return fmt.Errorf("listen address or mirror addresses are empty")
	}

	return nil
}
