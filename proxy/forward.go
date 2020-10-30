package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dantin/logger"
)

// connection represents an UDP connection with last activity timestamp.
type connection struct {
	udp          *net.UDPConn
	lastActivity time.Time
}

// packet represents an UDP packet payload with peer network address.
type packet struct {
	src  *net.UDPAddr
	data []byte
}

// Forwarder forward UDP packet from downstream to upstream.
type Forwarder struct {
	upstreamIP   string
	upstreamPort int

	connTimeout time.Duration
	resolveTTL  time.Duration

	client   *net.UDPAddr
	upstream *net.UDPAddr

	bufferPool sync.Pool
	closed     uint32
	connsMap   sync.Map

	upstreamMsgCh   chan packet
	downstreamMsgCh chan packet

	wg *sync.WaitGroup
}

// NewForwarder returns a new UDP forwarder.
func NewForwarder(wg *sync.WaitGroup, client, upstream *net.UDPAddr, connTimeout, resolveTTL time.Duration) *Forwarder {
	return &Forwarder{
		wg:              wg,
		client:          client,
		upstream:        upstream,
		upstreamIP:      upstream.IP.String(),
		upstreamPort:    upstream.Port,
		connTimeout:     connTimeout,
		resolveTTL:      resolveTTL,
		bufferPool:      sync.Pool{New: func() interface{} { return make([]byte, maxBufferSize) }},
		upstreamMsgCh:   make(chan packet),
		downstreamMsgCh: make(chan packet),
	}
}

// Run starts a forwarder.
func (fwd *Forwarder) Run(ctx context.Context) {
	logger.Debugf("start forward to upstream %s", fwd.upstream)
	atomic.StoreUint32(&fwd.closed, 0)

	go fwd.freeIdelSocketsLoop()
	go fwd.resolveUpstreamLoop()
	go fwd.handleDownstreamPackets()
	go fwd.handleUpstreamPackets()

	go func() {
		select {
		case <-ctx.Done():
			fwd.close()
		}
	}()
}

// Forward forwards a UDP packet to upstream.
func (fwd *Forwarder) Forward(pkt packet) {
	fwd.downstreamMsgCh <- pkt
}

func (fwd *Forwarder) close() {
	defer fwd.wg.Done()

	logger.Debugf("destroy forward to upstream %s", fwd.upstream)
	atomic.StoreUint32(&fwd.closed, 1)
	fwd.connsMap.Range(func(k, conn interface{}) bool {
		conn.(*connection).udp.Close()
		return true
	})
}

func (fwd *Forwarder) isClosed() bool {
	return atomic.LoadUint32(&fwd.closed) > 0
}

// handleDownstreamPackets forward UDP packet from downstream to upstream.
func (fwd *Forwarder) handleDownstreamPackets() {
	for pkt := range fwd.downstreamMsgCh {
		if fwd.isClosed() {
			break
		}

		clientAddr := pkt.src.String()

		conn, found := fwd.connsMap.Load(clientAddr)
		if !found {
			conn, err := net.ListenUDP("udp", fwd.client)
			if err != nil {
				logger.Warnf("udp forwarder failed to dail, drop packet, err %v", err)
				continue
			}
			fwd.connsMap.Store(clientAddr, &connection{
				udp:          conn,
				lastActivity: time.Now(),
			})

			conn.WriteTo(pkt.data, fwd.upstream)
			go fwd.downstreamReadLoop(pkt.src, conn)
		} else {
			conn.(*connection).udp.WriteTo(pkt.data, fwd.upstream)
			shouldUpdateLastActivity := false
			if conn, found := fwd.connsMap.Load(clientAddr); found {
				if conn.(*connection).lastActivity.Before(
					time.Now().Add(-fwd.connTimeout / 4)) {
					shouldUpdateLastActivity = true
				}
			}
			if shouldUpdateLastActivity {
				fwd.updateClientLastActivity(clientAddr)
			}
		}
	}
}

func (fwd *Forwarder) downstreamReadLoop(addr *net.UDPAddr, upstreamConn *net.UDPConn) {
	clientAddr := addr.String()
	for {
		if fwd.isClosed() {
			break
		}
		msg := fwd.bufferPool.Get().([]byte)
		size, _, err := upstreamConn.ReadFrom(msg[:])
		if err != nil {
			upstreamConn.Close()
			fwd.connsMap.Delete(clientAddr)
			return
		}
		fwd.updateClientLastActivity(clientAddr)
		fwd.upstreamMsgCh <- packet{
			src:  addr,
			data: msg[:size],
		}
	}
}

// handleUpstreamPackets handle response from upstream.
func (fwd *Forwarder) handleUpstreamPackets() {
	var respCnt uint64
	for pa := range fwd.upstreamMsgCh {
		if fwd.isClosed() {
			break
		}

		fwd.bufferPool.Put(pa.data)
		atomic.AddUint64(&respCnt, 1)
	}
}

func (fwd *Forwarder) updateClientLastActivity(clientAddr string) {
	if conn, found := fwd.connsMap.Load(clientAddr); found {
		conn.(*connection).lastActivity = time.Now()
	}
}

func (fwd *Forwarder) freeIdelSocketsLoop() {
	for {
		if fwd.isClosed() {
			break
		}

		time.Sleep(fwd.connTimeout)

		var (
			clientsToTimeout []string
			checkTimestamp   = time.Now().Add(-fwd.connTimeout)
		)

		fwd.connsMap.Range(func(k, conn interface{}) bool {
			if conn.(*connection).lastActivity.Before(checkTimestamp) {
				clientsToTimeout = append(clientsToTimeout, k.(string))
			}
			return true
		})

		for _, client := range clientsToTimeout {
			conn, ok := fwd.connsMap.Load(client)
			if ok {
				conn.(*connection).udp.Close()
				fwd.connsMap.Delete(client)
			}
		}
	}
}

func (fwd *Forwarder) resolveUpstreamLoop() {
	for {
		if fwd.isClosed() {
			break
		}

		time.Sleep(fwd.resolveTTL)
		upstreamAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", fwd.upstreamIP, fwd.upstreamPort))
		if err != nil {
			logger.Warnf("resovle upstream UDP address error, %v", err)
			continue
		}
		if upstreamAddr.String() != fwd.upstream.String() {
			logger.Infof("switch forward upstream from %s to %s", fwd.upstream, upstreamAddr)
			fwd.upstream = upstreamAddr
		}
	}
}
