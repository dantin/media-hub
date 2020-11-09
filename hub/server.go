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
        listen {{.Port}};
        latency 20;                          #ms

        domain_player live.sls.com;
        domain_publisher uplive.sls.com;
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
	confDir    = "conf"
	binDir     = "bin"
	slsName    = "sls"
	slsCfgName = "sls.conf"
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
	if cfg.SLSPath == "" {
		cfg.SLSPath = "sls.conf"
	}
	cfg.SLSPath = utils.ToAbsolutePath(rootpath, cfg.SLSPath)

	return &Server{cfg: cfg}
}

// Run runs SRT live server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// create PID file.
	if err := utils.CreatePIDFile(s.cfg.PIDFile); err != nil {
		return err
	}
	slsPath := filepath.Dir(s.cfg.SLSPath)

	s.setupSLSCfg(slsPath)

	return s.serve(slsPath, utils.SignalHandler())
}

// serve runs SRT living server.
func (s *Server) serve(slsHomePath string, stop <-chan bool) error {
	errCh := make(chan error)

	slsBinPath := filepath.Join(slsHomePath, binDir, slsName)
	slsCfgPath := filepath.Join(slsHomePath, confDir, slsCfgName)
	server := subprocess.NewSubprocess(errCh, slsBinPath, nil, "-c", slsCfgPath)

	if err := server.Run(); err != nil {
		logger.Warnf("SRT live server error, %v", err)
		return err
	}

	var relays []*subprocess.Subprocess
	for key, port := range s.cfg.PortRelayMap {
		srtListenOn := fmt.Sprintf("srt://:%d", port)
		uploadURL := fmt.Sprintf("srt://127.0.0.1:8080?streamid=uplive.sls.com/live/%s", key)
		relay := subprocess.NewSubprocess(errCh, "/usr/local/bin/srt-live-transmit", nil, srtListenOn, uploadURL)
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

func (s *Server) setupSLSCfg(basePath string) error {
	if basePath == "" {
		return fmt.Errorf("SLS base path is empty")
	}

	// Prepare SLS config directory.
	cfgDir := filepath.Join(basePath, confDir)
	if err := os.MkdirAll(cfgDir, os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(cfgDir, slsCfgName))
	if err != nil {
		return err
	}
	defer file.Close()

	// Setup SLS configuration content.
	buf := bytes.NewBuffer(nil)
	templ := template.Must(template.New("slsCfgTemplate").Parse(slsCfgTemplate))
	if err := templ.Execute(buf, struct {
		Port int
	}{
		Port: s.cfg.ListenAddr,
	}); err != nil {
		return err
	}

	if _, err := file.WriteString(buf.String()); err != nil {
		return err
	}

	return nil
}
