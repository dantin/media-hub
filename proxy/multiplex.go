package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dantin/logger"
	"github.com/dantin/media-hub/pkg/utils"
)

const maxBufferSize = 10 * (1 << 10) // 10K

// Multiplex encapsulates several UDP forwards which forward each UDP packet from its listening address to its forward list.
type Multiplex struct {
	listenAddr *net.UDPAddr
	forwards   []*Forwarder

	listenConn *net.UDPConn

	closed     uint32
	bufferPool sync.Pool
}

// NewMultiplex returns a runnable UDP multiplex using the given configuration.
func NewMultiplex(cfg *Config) *Multiplex {
	m := &Multiplex{
		listenAddr: cfg.ListenAddr,
		bufferPool: sync.Pool{New: func() interface{} { return make([]byte, maxBufferSize) }},
	}

	// build UDP forwards.
	var forwards []*Forwarder
	for _, ma := range cfg.MirrorAddrs {
		client := &net.UDPAddr{
			IP:   cfg.ListenAddr.IP,
			Port: 0,
			Zone: cfg.ListenAddr.Zone,
		}
		upstream, err := net.ResolveUDPAddr("udp", ma.String())
		if err != nil {
			logger.Warnf("Resovle upstream UDP address for item '%s', error, %v", ma, err)
			continue
		}
		forwards = append(forwards, NewForwarder(client, upstream, cfg.ConnectTimeout, cfg.ResolveTTL))
	}
	if len(forwards) == 0 {
		logger.Warnf("UDP multiplex will run without upstream service")
	}
	m.forwards = forwards

	return m
}

// Run runs UDP multiplex server until either a stop signal is received or an error occurs.
func (m *Multiplex) Run() error {
	var wg sync.WaitGroup

	shuttingDown := false
	stop := utils.SignalHandler()
	done := make(chan bool)

	// run forwards.
	for _, fwd := range m.forwards {
		fwd.Run()
		wg.Add(1)
	}

	go func() {
		var err error
		logger.Infof("Listen for client UDP connections on [%s]", m.listenAddr)
		err = m.serverLoop()
		if err != nil {
			if shuttingDown {
				logger.Infof("UDP multiplex: stopped")
			} else {
				logger.Warnf("UDP multiplex: failed", err)
			}
		}
		done <- true
	}()

	// Wait for either a termination signal or an error
Loop:
	for {
		select {
		case <-stop:
			shuttingDown = true
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := m.Close(ctx, &wg); err != nil {
				// Failure/timeout shutting down the multiplex gracefully.
				logger.Warnf("UDP multiplex failed to terminate gracefully %s", err)
			}

			// Wait for
			<-done
			cancel()

			break Loop
		case <-done:
			break Loop
		}
	}

	return nil
}

// Close close multiplex.
func (m *Multiplex) Close(ctx context.Context, wg *sync.WaitGroup) error {
	atomic.StoreUint32(&m.closed, 1)

	// close forwards.
	for _, fwd := range m.forwards {
		fwd.Close()
		wg.Done()
	}
	// wait all forwards closed.
	wg.Wait()

	if m.listenConn != nil {
		m.listenConn.Close()
	}

	return nil
}

func (m *Multiplex) serverLoop() error {
	conn, err := net.ListenUDP("udp", m.listenAddr)
	if err != nil {
		return fmt.Errorf("error while listening on bind port: %s", err)
	}
	m.listenConn = conn

	logger.Infof("UDP multiplex is listening on %s", m.listenAddr)

	for {
		if atomic.LoadUint32(&m.closed) > 0 {
			break
		}
		msg := m.bufferPool.Get().([]byte)
		size, srcAddr, err := m.listenConn.ReadFromUDP(msg[:])
		if err != nil {
			continue
		}

		for _, fwd := range m.forwards {
			fwd.Forward(packet{
				src:  srcAddr,
				data: msg[:size],
			})
		}
		m.bufferPool.Put(msg)
	}

	return nil
}
