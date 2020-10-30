package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dantin/logger"
)

const (
	version = "0.3.0-dev"
)

var (
	listenAddr string

	showVersion bool
	showUsage   bool
)

func parseArgs(args []string) error {
	var (
		configFile string
		apiPath    string
		expvarPath string
		pprofFile  string
		pprofURL   string
		level      string
	)

	executable, _ := os.Executable()
	_, appName := filepath.Split(executable)

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.StringVar(&configFile, "config", "config.yml", "Path to config file.")
	fs.StringVar(&listenAddr, "listen", "", "Override addess and port to listen on for HTTP clients.")
	fs.StringVar(&apiPath, "api_path", "", "Override the base URL path where API is served.")
	fs.StringVar(&expvarPath, "expvar", "", "Override the URL path where runtime stats are exposed. Use '-' to disable.")
	fs.StringVar(&pprofFile, "pprof", "", "File name to save profiling info to. Disable if not set.")
	fs.StringVar(&pprofURL, "pprof_url", "", "Debugging only! URL path for exposing profiling info. Disable if not set.")
	fs.StringVar(&level, "level", "info", "Log level, supported level: debug, info, error, fatal.")
	fs.BoolVar(&showVersion, "v", false, "Print version information.")
	fs.BoolVar(&showUsage, "h", false, "Show help message.")

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

	l, err := logger.New(level, os.Stdout)
	if err != nil {
		return fmt.Errorf("fail to resovle bind address, %v", err)
	}
	logger.Set(l)

	return nil
}

func main() {
	l, err := logger.New("debug", os.Stdout)
	if err != nil {
		panic(err)
	}
	logger.Set(l)

	defer logger.Unset()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sc
		fmt.Printf("%s - Shutdown signal received...\n", sig)
	}()

	executable, _ := os.Executable()
	rootPath, appName := filepath.Split(executable)
	logger.Infof("%s running on %s", appName, rootPath)

	logger.Debugf("haha")
}
