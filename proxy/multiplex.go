package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/dantin/logger"
)

const maxBufferSize = 10 * (1 << 10) // 10K

// Multiplex encapsulates several UDP forwards which forward each UDP packet from its listening address to its forward list.
type Multiplex struct {
	listenAddr *net.UDPAddr
	forwards   []*Forwarder

	listenConn *net.UDPConn

	closed     uint32
	bufferPool sync.Pool
	wg         sync.WaitGroup
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
		upstream, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ma.ipAddr, ma.port))
		if err != nil {
			logger.Warnf("resovle upstream UDP address for item %s:%d, error, %v", ma.ipAddr, ma.port, err)
			continue
		}
		forwards = append(forwards, NewForwarder(&m.wg, client, upstream, cfg.ConnectTimeout, cfg.ResolveTTL))
	}
	if len(forwards) == 0 {
		logger.Warnf("UDP multiplex will run in NO forwarding mode")
	}
	m.forwards = forwards

	return m
}

// Run runs UDP multiplex server until either a stop signal is received or an error occurs.
func (m *Multiplex) Run(ctx context.Context) error {
	// run forwards.
	for _, fwd := range m.forwards {
		fwd.Run(ctx)
		m.wg.Add(1)
	}

	go func() {
		select {
		case <-ctx.Done():
			m.Close()
		}
	}()

	return m.serverLoop(ctx)
}

// Close close multiplex.
func (m *Multiplex) Close() {
	// wait forwards closing.
	m.wg.Wait()

	atomic.StoreUint32(&m.closed, 1)

	if m.listenConn != nil {
		m.listenConn.Close()
	}
}

func (m *Multiplex) serverLoop(ctx context.Context) error {
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
