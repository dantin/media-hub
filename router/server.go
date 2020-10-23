package router

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

const maxBufferSize = 1024

type mirror struct {
	addr   string
	conn   *net.UDPConn
	closed uint32
}

// Server encapsulates a UDP multiplex server application.
type Server struct {
	listenAddr  string
	mirrorAddrs mirrorList

	watchStopCh chan os.Signal
}

// NewServer returns a runnable UDP multiplex server given a command line arguments array.
func NewServer(cfg *Config) *Server {
	return &Server{
		listenAddr:  cfg.ListenAddr,
		mirrorAddrs: cfg.MirrorAddrs,
	}
}

// Run runs UDP multiplex server until either a stop signal is received or an error occurs.
func (s *Server) Run() error {
	s.watchStopCh = make(chan os.Signal, 1)
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
	case s := <-s.watchStopCh:
		log.Printf("signal %v received, waiting for server to exit.", s)
		cancel()
		log.Printf("exiting...")
		return nil
	}
}

func (s *Server) serverLoop(ctx context.Context) {
	pc, err := net.ListenPacket("udp", s.listenAddr)
	if err != nil {
		log.Fatalf("error while listening: %s", err)
	}
	defer pc.Close()

	log.Printf("UDP multiplex server is listening on %s", s.listenAddr)

	var done uint32
	go func() {
		select {
		case <-ctx.Done():
			atomic.AddUint32(&done, 1)
		}
	}()

	for {
		if atomic.LoadUint32(&done) > 0 {
			return
		}
		doneCh := make(chan error, 1)

		go s.relay(ctx, pc, doneCh)

		select {
		case err := <-doneCh:
			if err != nil {
				log.Printf("error occured while handle recevied packet")
			}
		}
	}
}

func (s *Server) relay(ctx context.Context, pc net.PacketConn, doneCh chan error) {
	var mirrors []mirror

	for _, addr := range s.mirrorAddrs {
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Printf("error while connecting to mirror %s (%s), will continur", addr, err)
			continue
		}

		mirrors = append(mirrors, mirror{
			addr:   addr.String(),
			conn:   conn,
			closed: 0,
		})
	}

	defer func() {
		for i, m := range mirrors {
			if closed := atomic.LoadUint32(&mirrors[i].closed); closed == 1 {
				continue
			}
			m.conn.Close()
		}
	}()

	closeCh := make(chan error, 1024)
	errorCh := make(chan error, 1024)

	go connect(ctx, pc, mirrors, closeCh, errorCh)

	for {
		select {
		case err := <-errorCh:
			if err != nil {
				log.Printf("got error (%s), will continue", err)
			}
		case err := <-closeCh:
			if err != nil {
				log.Printf("got error (%s), will close client connection", err)
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func connect(ctx context.Context, origin net.PacketConn, mirrors []mirror, closeCh, errorCh chan error) {
	for i := 0; i < len(mirrors); i++ {
		go readAndDiscard(ctx, mirrors[i], errorCh)
	}

	go forwardAndCopy(ctx, origin, mirrors, closeCh, errorCh)
}

func readAndDiscard(ctx context.Context, m mirror, closeCh chan error) {
	var done uint32
	go func() {
		select {
		case <-ctx.Done():
			atomic.AddUint32(&done, 1)
		}
	}()

	for {
		if atomic.LoadUint32(&done) > 0 {
			return
		}
		var b [maxBufferSize]byte

		_, _, err := m.conn.ReadFrom(b[:])
		if err != nil {
			m.conn.Close()
			atomic.StoreUint32(&m.closed, 1)
			select {
			case closeCh <- err:
			default:
			}
			return
		}
		//log.Printf("packet-received: bytes=%d from=%s", n, addr.String())
	}
}

func forwardAndCopy(ctx context.Context, src net.PacketConn, mirrors []mirror, closeCh, errorCh chan error) {
	var done uint32
	go func() {
		select {
		case <-ctx.Done():
			atomic.AddUint32(&done, 1)
		}
	}()

	for {
		if atomic.LoadUint32(&done) > 0 {
			return
		}

		var b [maxBufferSize]byte

		n, _, err := src.ReadFrom(b[:])
		if err != nil {
			closeCh <- err
			return
		}

		//log.Printf("packet-received: bytes=%d from=%s", n, addr.String())

		for i := 0; i < len(mirrors); i++ {
			if closed := atomic.LoadUint32(&mirrors[i].closed); closed == 1 {
				continue
			}

			mirrors[i].conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err != nil {
				closeCh <- err
				return
			}

			n, err = mirrors[i].conn.Write(b[:n])
			if err != nil {
				mirrors[i].conn.Close()
				atomic.StoreUint32(&mirrors[i].closed, 1)
				select {
				case errorCh <- err:
				default:
				}
			}
		}
		//log.Printf("packet-forward: bytes=%d", n)
	}
}
