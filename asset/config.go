package asset

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dantin/logger"
	yaml "gopkg.in/yaml.v2"
)

const (
	version     = "0.3.0-dev"
	defaultName = "asset-server"
)

// Config holds configuration of proxy.
type Config struct {
	*flag.FlagSet

	PIDFile    string `yaml:"pid_file"`
	ListenAddr string `yaml:"listen"`
	APIPath    string `yaml:"api_path"`
	ExpvarPath string `yaml:"expvar_path"`
	PProfFile  string `yaml:"pprof"`
	PProfURL   string `yaml:"pprof_url"`
}

// NewConfig creates an instance of UDP mutiplex configuration.
func NewConfig() *Config {
	return &Config{}
}

// Parse parses configuration from command line arguments.
func (cfg *Config) Parse(args []string) error {
	var (
		configFile  string
		level       string
		showVersion bool
		showUsage   bool
	)
	executable, _ := os.Executable()
	_, appName := filepath.Split(executable)

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.StringVar(&configFile, "config", "config.yml", "Path to config file.")
	fs.StringVar(&cfg.ListenAddr, "listen", "", "Override addess and port to listen on for HTTP clients.")
	fs.StringVar(&cfg.APIPath, "api_path", "", "Override the base URL path where API is served.")
	fs.StringVar(&cfg.ExpvarPath, "expvar", "", "Override the URL path where runtime stats are exposed. Use '-' to disable.")
	fs.StringVar(&cfg.PProfFile, "pprof", "", "File name to save profiling info to. Disable if not set.")
	fs.StringVar(&cfg.PProfURL, "pprof_url", "", "Debugging only! URL path for exposing profiling info. Disable if not set.")
	fs.StringVar(&level, "level", "info", "Log level, supported level: debug, info, error, fatal.")
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.BoolVar(&showUsage, "h", false, "Show help message.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if showVersion {
		fmt.Printf("%s %s\n", appName, version)
		os.Exit(0)
	}

	if showUsage {
		fs.Usage()
		os.Exit(0)
	}

	l, err := logger.New(level, os.Stdout)
	if err != nil {
		return fmt.Errorf("fail to setup logger, %v", err)
	}
	logger.Set(l)

	// load configuration if specified.
	if configFile != "" {
		logger.Infof("Using config file from '%s'", configFile)
		if err := cfg.configFromFile(configFile); err != nil {
			return fmt.Errorf("fail to load config from file, %v", err)
		}
	}

	// parse again to replace config with command line options.
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) > 0 {
		return fmt.Errorf("%q is not a valid flag", fs.Arg(0))
	}

	return nil
}

func (cfg *Config) configFromFile(path string) error {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFile, cfg)
}

func (cfg *Config) String() {
	return
}
