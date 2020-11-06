package asset

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/pkg/utils"
)

const defaultAPIPath = "/"

// Server encapsulates a HTTP server which provide asset related information.
type Server struct {
	cfg *Config
}

// NewServer returns a runnable HTTP server using the given configuration.
func NewServer(cfg *Config) *Server {
	executable, _ := os.Executable()
	rootpath, _ := filepath.Split(executable)

	cfg.PProfFile = utils.ToAbsolutePath(rootpath, cfg.PProfFile)
	cfg.PIDFile = utils.ToAbsolutePath(rootpath, cfg.PIDFile)

	// normalize API path.
	if cfg.APIPath == "" {
		cfg.APIPath = defaultAPIPath
	} else {
		if !strings.HasPrefix(cfg.APIPath, "/") {
			cfg.APIPath = "/" + cfg.APIPath
		}
		if !strings.HasSuffix(cfg.APIPath, "/") {
			cfg.APIPath += "/"
		}
	}
	return &Server{cfg: cfg}
}

// Run runs HTTP server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// create PID file.
	if err := utils.CreatePIDFile(s.cfg.PIDFile); err != nil {
		return err
	}

	// set up HTTP server. Must use non-default mux because of expvar.
	mux := http.NewServeMux()

	// exposing values for statistics and monitoring.
	statsInit(mux, s.cfg.ExpvarPath)

	// initialize serving debug profiles (optional).
	servePprof(mux, s.cfg.PProfURL)

	if s.cfg.PProfFile != "" {
		cpuf, err := os.Create(s.cfg.PProfFile + ".cpu")
		if err != nil {
			logger.Fatalf("Fail to create CPU pprof file: %v", err)
		}
		defer cpuf.Close()

		memf, err := os.Create(s.cfg.PProfFile + ".mem")
		if err != nil {
			logger.Fatalf("Fail to create MEM pprof file: %v", err)
		}
		defer memf.Close()

		pprof.StartCPUProfile(cpuf)
		defer pprof.StopCPUProfile()
		defer pprof.WriteHeapProfile(memf)

		logger.Infof("Profiling info saved to '%s.(cpu|mem)'", s.cfg.PProfFile)
	}

	// configure root path for serving API calls.
	logger.Infof("API served from root URL path '%s'", s.cfg.APIPath)
	mux.HandleFunc(s.cfg.APIPath+"v0/index", index)

	return listenAndServe(s.cfg.ListenAddr, mux, utils.SignalHandler())
}
