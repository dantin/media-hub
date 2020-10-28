package proxy

import (
	"context"
	"fmt"
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
	wg          sync.WaitGroup
}

// NewServer returns a runnable UDP multiplex server given a command line arguments array.
func NewServer(cfg *Config) *Server {
	listenAddr, err := net.ResolveUDPAddr("udp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("fail to resovle bind address, %v", err)
	}
	svr := &Server{
		listenAddr: listenAddr,
		bufferPool: sync.Pool{New: func() interface{} { return make([]byte, maxBufferSize) }},
	}

	// build UDP forwards.
	var forwards []*Forwarder
	for _, ma := range cfg.MirrorAddrs {
		client := &net.UDPAddr{
			IP:   listenAddr.IP,
			Port: 0,
			Zone: listenAddr.Zone,
		}
		upstream, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ma.ipAddr, ma.port))
		if err != nil {
			log.Printf("resovle upstream UDP address for item %s:%d, error, %v", ma.ipAddr, ma.port, err)
			continue
		}
		forwards = append(forwards, NewForwarder(&svr.wg, client, upstream, cfg.ConnectTimeout, cfg.ResolveTTL, maxBufferSize))
	}
	if len(forwards) == 0 {
		log.Printf("server will run in NO forwarding mode")
	}
	svr.forwards = forwards

	return svr
}

// Run runs UDP multiplex server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	// setup shutdown handler.
	s.watchStopCh = make(chan os.Signal, 1)
	signal.Notify(s.watchStopCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// run server loop.
	ctx, cancel := context.WithCancel(context.Background())
	go s.serverLoop(ctx)

	// run forward.
	for _, fwd := range s.forwards {
		fwd.Run(ctx)
		s.wg.Add(1)
	}

	select {
	case sig := <-s.watchStopCh:
		log.Printf("signal %v received, waiting for server to exit.", sig)
		cancel()
		s.wg.Wait()
		log.Printf("exiting...")
		return nil
	}
}

func (s *Server) serverLoop(ctx context.Context) {
	listenConn, err := net.ListenUDP("udp", s.listenAddr)
	if err != nil {
		log.Fatalf("error while listening on bind port: %s", err)
	}
	defer listenConn.Close()

	log.Printf("UDP multiplex is listening on %s", s.listenAddr)

	for {
		msg := s.bufferPool.Get().([]byte)
		size, srcAddr, err := listenConn.ReadFromUDP(msg[:])
		if err != nil {
			log.Printf("read UDP packet error: %v", err)
			continue
		}

		for _, fwd := range s.forwards {
			fwd.Forward(packet{
				src:  srcAddr,
				data: msg[:size],
			})
		}
		s.bufferPool.Put(msg)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
