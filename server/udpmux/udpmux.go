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
	params      *Params
	conns       map[string]*muxedConn
	mu          sync.Mutex
	logger      logger.Logger
	debugLogger logger.Logger
	connChan    chan Conn
	closeChan   chan struct{}
	closeOnce   sync.Once
	wg          sync.WaitGroup
}

type Params struct {
	Conn          net.PacketConn
	MTU           uint32
	LoggerFactory logger.LoggerFactory
	ReadChanSize  int
}

func New(params Params) *UDPMux {
	u := &UDPMux{
		params:      &params,
		conns:       map[string]*muxedConn{},
		logger:      params.LoggerFactory.GetLogger("udpmux:info"),
		debugLogger: params.LoggerFactory.GetLogger("udpmux:debug"),
		connChan:    make(chan Conn),
		closeChan:   make(chan struct{}),
	}

	if u.params.MTU == 0 {
		u.params.MTU = DefaultMTU
	}

	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.start()
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

	u.logger.Printf("%s AcceptConn", conn, conn.RemoteAddr())

	return conn, nil
}

func (u *UDPMux) GetConn(raddr net.Addr) (Conn, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	select {
	case <-u.closeChan:
		return nil, fmt.Errorf("UDPMux closed")
	default:
	}

	if _, ok := u.conns[raddr.String()]; ok {
		return nil, fmt.Errorf("Connection already exists")
	}

	c := u.createConn(raddr, false)

	u.logger.Printf("%s GetConn", c, raddr)

	return c, nil
}

func (u *UDPMux) start() {
	buf := make([]byte, u.params.MTU)

	for {
		i, raddr, err := u.params.Conn.ReadFrom(buf)

		if err != nil {
			u.logger.Printf("Error reading remote data: %s", err)
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
	u.logger.Println("Close")

	err := u.params.Conn.Close()

	u.close()

	u.wg.Wait()

	return err
}

func (u *UDPMux) close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, conn := range u.conns {
		u.closeConn(conn)
	}

	u.closeOnce.Do(func() {
		close(u.connChan)
		close(u.closeChan)
	})
}

func (u *UDPMux) handleClose(conn *muxedConn) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.closeConn(conn)
}

func (u *UDPMux) closeConn(conn *muxedConn) {
	u.logger.Printf("%s closeConn", conn)

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
	select {
	case <-c.closeChan:
		u.logger.Println("Ignoring remote data because connection has been closed")
		return
	default:
	}

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
		debugLogger: u.debugLogger,
		conn:        u.params.Conn,
		laddr:       u.params.Conn.LocalAddr(),
		raddr:       raddr,
		readChan:    make(chan []byte, u.params.ReadChanSize),
		closeChan:   make(chan struct{}),
		onClose:     u.handleClose,
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
	conn        net.PacketConn
	laddr       net.Addr
	raddr       net.Addr
	readChan    chan []byte
	closeChan   chan struct{}
	onClose     func(m *muxedConn)
	onceClose   sync.Once
	debugLogger logger.Logger
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
	m.debugLogger.Printf("%s recv %v", m, buf)
	return len(buf), nil
}

func (m *muxedConn) Write(b []byte) (int, error) {
	select {
	case <-m.closeChan:
		return 0, fmt.Errorf("Conn is closed")
	default:
		m.debugLogger.Printf("%s send %v", m, b)
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

func (m *muxedConn) String() string {
	if s, ok := m.conn.(stringer); ok {
		return fmt.Sprintf("%s [%s %s]", s.String(), m.laddr, m.raddr)
	}

	return fmt.Sprintf("[%s %s]", m.laddr, m.raddr)
}

var _ net.Conn = &muxedConn{}

type stringer interface {
	String() string
}
