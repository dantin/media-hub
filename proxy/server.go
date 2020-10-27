package proxy

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const maxBufferSize = 10 * 1024

// Server encapsulates a UDP multiplex server application.
type Server struct {
	listenAddr *net.UDPAddr
	forwards   []*Forwarder

	bufferPool  sync.Pool
	watchStopCh chan os.Signal
}

// NewServer returns a runnable UDP multiplex server given a command line arguments array.
func NewServer(cfg *Config) *Server {
	listenAddr, err := net.ResolveUDPAddr("udp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("fail to resovle bind address, %v", err)
	}
	var forwards []*Forwarder
	for _, ma := range cfg.MirrorAddrs {
		fwd, err := NewForwarder(listenAddr.IP, listenAddr.Zone, ma.ipAddr, ma.port, cfg.ConnectTimeout, cfg.ResolveTTL, maxBufferSize)
		if err != nil {
			log.Printf("fail to build forwarder to %s:%d, ignore", ma.ipAddr, ma.port)
			continue
		}
		forwards = append(forwards, fwd)
	}
	if len(forwards) == 0 {
		log.Printf("server will run without any forward")
	}
	return &Server{
		listenAddr:  listenAddr,
		forwards:    forwards,
		bufferPool:  sync.Pool{New: func() interface{} { return make([]byte, maxBufferSize) }},
		watchStopCh: make(chan os.Signal, 1),
	}
}

// Run runs UDP multiplex server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// setup shutdown handler.
	signal.Notify(s.watchStopCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// run server loop.
	ctx, cancel := context.WithCancel(context.Background())
	go s.serverLoop(ctx)

	select {
	case ss := <-s.watchStopCh:
		log.Printf("signal %v received, waiting for server to exit.", ss)
		cancel()
		s.Close()
		log.Printf("exiting...")
		return nil
	}
}

// Close shutdown forwards.
func (s *Server) Close() {
	for _, fwd := range s.forwards {
		fwd.Close()
	}
}

func (s *Server) serverLoop(ctx context.Context) {
	listenConn, err := net.ListenUDP("udp", s.listenAddr)
	if err != nil {
		log.Fatalf("error while listening on bind port: %s", err)
	}
	defer listenConn.Close()

	log.Printf("UDP multiplex server is listening on %s", s.listenAddr)
	for _, fwd := range s.forwards {
		fwd.Run()
	}

	for {
		msg := s.bufferPool.Get().([]byte)
		size, srcAddr, err := listenConn.ReadFromUDP(msg[:])
		if err != nil {
			log.Printf("read UDP packet error: %v", err)
			continue
		}

		log.Printf("got UDP packet from %s, size %d", srcAddr, size)

		for _, fwd := range s.forwards {
			fwd.Forward(packet{
				src:  srcAddr,
				data: msg[:size],
			})
		}

		select {
		case <-ctx.Done():
			log.Printf("quit server loop")
			return
		default:
		}
	}
}
