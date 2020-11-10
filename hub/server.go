package hub

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/pkg/utils"
	"github.com/dantin/media-hub/subprocess"
)

const (
	slsCfgTemplate = `
# SRT configuration template
srt {
    worker_threads  1;
    worker_connections 300;

    log_file logs/error.log;
    log_level info;

    record_hls_path_prefix /tmp/mov/sls;

    server {
        listen {{.ListenOn}};
        latency 20;                          #ms

        domain_player {{.Domain}};
        domain_publisher up{{.Domain}};
        backlog 100;                         #accept connections at the same time
        idle_streams_timeout 10;             #s -1: unlimited
        app {
            app_player live;
            app_publisher live;

            record_hls off;                  #on, off
            record_hls_segment_duration 10;  #unit s
        }
    }
}
`
)

// Server encapsulates a SRT live server.
type Server struct {
	cfg *Config
}

// NewServer returns a runnable SRT live server using the given configuration.
func NewServer(cfg *Config) *Server {
	executable, _ := os.Executable()
	rootpath, _ := filepath.Split(executable)

	if cfg.PIDFile == "" {
		cfg.PIDFile = "srt-server.pid"
	}
	cfg.PIDFile = utils.ToAbsolutePath(rootpath, cfg.PIDFile)
	if cfg.HomePath == "" {
		cfg.HomePath = rootpath
	}
	cfg.HomePath = utils.ToAbsolutePath(rootpath, cfg.HomePath)

	return &Server{cfg: cfg}
}

// Run runs SRT live server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// create PID file.
	if err := utils.CreatePIDFile(s.cfg.PIDFile); err != nil {
		return err
	}

	s.setupSLSCfg()

	return s.serve(utils.SignalHandler())
}

// serve runs SRT living server.
func (s *Server) serve(stop <-chan bool) error {
	errCh := make(chan error)

	server := subprocess.NewSubprocess(errCh,
		filepath.Join(s.cfg.HomePath, "bin", "sls"),
		nil,
		"-c",
		filepath.Join(s.cfg.HomePath, "conf", "sls.conf"))

	if err := server.Run(); err != nil {
		logger.Warnf("SRT live server error, %v", err)
		return err
	}

	var relays []*subprocess.Subprocess
	for key, port := range s.cfg.PortRelayMap {
		relay := subprocess.NewSubprocess(errCh,
			"/usr/local/bin/srt-live-transmit",
			nil,
			fmt.Sprintf("srt://:%d", port),
			fmt.Sprintf("srt://127.0.0.1:%d?streamid=up%s/live/%s", s.cfg.SRTCfg.ListenOn, s.cfg.SRTCfg.Domain, key))
		relays = append(relays, relay)
	}

	logger.Infof("There are %d port relay is ready to run.", len(relays))

	for _, relay := range relays {
		if err := relay.Run(); err != nil {
			logger.Warnf("srt-live-transmit start error, %v", err)
			continue
		}
	}

	// wait for either a termination signal or an underlying error happens.
Loop:
	for {
		select {
		case <-stop:
			// give server 2 seconds to shut down.
			_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := server.Stop(); err != nil {
				logger.Warnf("SRT live server failed to terminate gracefully, %v", err)
			}
			for _, relay := range relays {
				if err := relay.Stop(); err != nil {
					logger.Warnf("srt-live-transmit stop error, %v", err)
					continue
				}
			}

			cancel()

			break Loop
		case err := <-errCh:
			logger.Warnf("Error from SRT live server, %v", err)
		}
	}

	return nil
}

func (s *Server) setupSLSCfg() error {
	if s.cfg.HomePath == "" {
		return fmt.Errorf("home path is empty")
	}

	// Prepare SLS config directory.
	cfgDir := filepath.Join(s.cfg.HomePath, "conf")
	if err := os.MkdirAll(cfgDir, os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(cfgDir, "sls.conf"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Setup SLS configuration content.
	buf := bytes.NewBuffer(nil)
	templ := template.Must(template.New("slsTemplate").Parse(slsCfgTemplate))
	if err := templ.Execute(buf, s.cfg.SRTCfg); err != nil {
		return err
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
