package hub

import (
	"os"
	"path/filepath"

	"github.com/dantin/media-hub/pkg/utils"
)

// Server encapsulates a SRT live server.
type Server struct {
	cfg *Config
}

// NewServer returns a runnable SRT live server using the given configuration.
func NewServer(cfg *Config) *Server {
	executable, _ := os.Executable()
	rootpath, _ := filepath.Split(executable)

	cfg.PIDFile = utils.ToAbsolutePath(rootpath, cfg.PIDFile)

	return &Server{cfg: cfg}
}

// Run runs SRT live server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// create PID file.
	if err := utils.CreatePIDFile(s.cfg.PIDFile); err != nil {
		return err
	}

	return listenAndServe(s.cfg.ListenAddr, utils.SignalHandler())
}
