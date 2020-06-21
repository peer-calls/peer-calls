package udpmux

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/peer-calls/peer-calls/server/logger"
)

var DefaultMTU uint32 = 8192

type UDPMux struct {
	params    *Params
	conns     map[string]*muxedConn
	mu        sync.Mutex
	logger    logger.Logger
	pktLogger logger.Logger
	connChan  chan Conn
	closeChan chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

type Params struct {
	Conn          net.PacketConn
	MTU           uint32
	LoggerFactory logger.LoggerFactory
	ReadChanSize  int
}

func New(params Params) *UDPMux {
	u := &UDPMux{
		params:    &params,
		conns:     map[string]*muxedConn{},
		logger:    params.LoggerFactory.GetLogger("udpmux"),
		pktLogger: params.LoggerFactory.GetLogger("udpmux:packets"),
		connChan:  make(chan Conn),
		closeChan: make(chan struct{}),
	}

	if u.params.MTU == 0 {
		u.params.MTU = DefaultMTU
	}

	u.wg.Add(1)
	go func() {
		u.start()
		u.wg.Done()
	}()

	return u
}

func (u *UDPMux) LocalAddr() net.Addr {
	return u.params.Conn.LocalAddr()
}

func (u *UDPMux) AcceptConn() (Conn, error) {
	conn, ok := <-u.connChan
	if !ok {
		return nil, fmt.Errorf("Conn closed")
	}
	u.pktLogger.Printf("[%s <- %s] AcceptConn", u.params.Conn.LocalAddr(), conn.RemoteAddr())
	return conn, nil
}

func (u *UDPMux) GetConn(raddr net.Addr) (Conn, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.pktLogger.Printf("[%s -> %s] GetConn", u.params.Conn.LocalAddr(), raddr)
	// TODO return err when not connected
	return u.getOrCreateConn(raddr, false), nil
}

func (u *UDPMux) start() {
	buf := make([]byte, u.params.MTU)

	for {
		i, raddr, err := u.params.Conn.ReadFrom(buf)

		if err != nil {
			u.logger.Println("Error reading remote data: %s", err)
			_ = u.params.Conn.Close()
			return
		}

		u.handleRemoteBytes(raddr, buf[:i])
	}
}

func (u *UDPMux) CloseChannel() <-chan struct{} {
	return u.closeChan
}

func (u *UDPMux) Close() error {
	u.close()

	u.wg.Wait()

	return nil
}

func (u *UDPMux) close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, conn := range u.conns {
		conn.onceClose.Do(func() {
			close(conn.readChan)
			close(conn.closeChan)
		})
		delete(u.conns, conn.RemoteAddr().String())
	}

	u.closeOnce.Do(func() {
		close(u.closeChan)
		close(u.connChan)
		_ = u.params.Conn.Close()
	})
}

func (u *UDPMux) handleClose(conn *muxedConn) {
	u.mu.Lock()
	defer u.mu.Unlock()

	conn.onceClose.Do(func() {
		close(conn.readChan)
		close(conn.closeChan)
	})
	delete(u.conns, conn.RemoteAddr().String())
}

func (u *UDPMux) handleRemoteBytes(raddr net.Addr, buf []byte) {
	u.mu.Lock()
	defer u.mu.Unlock()

	select {
	case <-u.closeChan:
		u.logger.Println("Ignoring remote data because connection has been closed")
		return
	default:
	}

	c := u.getOrCreateConn(raddr, true)
	c.handleRemoteBytes(buf)
}

func (u *UDPMux) getOrCreateConn(raddr net.Addr, accept bool) *muxedConn {
	c, ok := u.conns[raddr.String()]
	if !ok {
		c = u.createConn(raddr, accept)
	}
	return c
}

func (u *UDPMux) createConn(raddr net.Addr, accept bool) *muxedConn {
	c := &muxedConn{
		pktLogger: u.pktLogger,
		conn:      u.params.Conn,
		laddr:     u.params.Conn.LocalAddr(),
		raddr:     raddr,
		readChan:  make(chan []byte, u.params.ReadChanSize),
		closeChan: make(chan struct{}),
		onClose:   u.handleClose,
	}
	u.conns[raddr.String()] = c
	if accept {
		u.connChan <- c
	}
	return c
}

type Conn interface {
	net.Conn
	CloseChannel() <-chan struct{}
}

type muxedConn struct {
	conn      net.PacketConn
	laddr     net.Addr
	raddr     net.Addr
	readChan  chan []byte
	closeChan chan struct{}
	onClose   func(m *muxedConn)
	onceClose sync.Once
	pktLogger logger.Logger
}

var _ Conn = &muxedConn{}

func (m *muxedConn) Close() error {
	m.onClose(m)
	return nil
}

func (m *muxedConn) handleRemoteBytes(buf []byte) {
	b := make([]byte, len(buf))
	copy(b, buf)
	m.readChan <- b
}

func (m *muxedConn) CloseChannel() <-chan struct{} {
	return m.closeChan
}

func (m *muxedConn) Read(b []byte) (int, error) {
	buf, ok := <-m.readChan
	if !ok {
		return 0, fmt.Errorf("Conn closed")
	}
	copy(b, buf)
	m.pktLogger.Printf("[%s <- %s] %v", m.laddr, m.raddr, buf)
	return len(buf), nil
}

func (m *muxedConn) Write(b []byte) (int, error) {
	select {
	case <-m.closeChan:
		return 0, fmt.Errorf("Conn is closed")
	default:
		m.pktLogger.Printf("[%s -> %s] %v", m.laddr, m.raddr, b)
		return m.conn.WriteTo(b, m.raddr)
	}
}

func (m *muxedConn) LocalAddr() net.Addr {
	return m.laddr
}

func (m *muxedConn) RemoteAddr() net.Addr {
	return m.raddr
}

func (m *muxedConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *muxedConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *muxedConn) SetWriteDeadline(t time.Time) error {
	return nil
}

var _ net.Conn = &muxedConn{}
