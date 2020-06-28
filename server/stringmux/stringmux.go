package stringmux

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/peer-calls/peer-calls/server/logger"
)

var DefaultMTU uint32 = 8192

type StringMux struct {
	params      *Params
	logger      logger.Logger
	debugLogger logger.Logger
	conns       map[string]*conn
	connChan    chan Conn
	closeChan   chan struct{}
	closeOnce   sync.Once
	mu          sync.Mutex
	wg          sync.WaitGroup
}

type Params struct {
	LoggerFactory logger.LoggerFactory
	Conn          net.Conn
	MTU           uint32
	ReadChanSize  int
}

func New(params Params) *StringMux {
	sm := &StringMux{
		params:      &params,
		logger:      params.LoggerFactory.GetLogger("stringmux:info"),
		debugLogger: params.LoggerFactory.GetLogger("stringmux:debug"),
		closeChan:   make(chan struct{}),
		connChan:    make(chan Conn),
		conns:       make(map[string]*conn),
	}

	if sm.params.MTU == 0 {
		sm.params.MTU = DefaultMTU
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.start()
	}()

	return sm
}

func (sm *StringMux) start() {
	buf := make([]byte, sm.params.MTU)

	for {
		i, err := sm.params.Conn.Read(buf)

		if err != nil {
			sm.logger.Printf("Error reading remote data: %s", err)
			_ = sm.params.Conn.Close()
			return
		}

		streamID, data, err := Unmarshal(buf[:i])
		if err != nil {
			sm.logger.Printf("Error unmarshaling remote data: %s", err)
			return
		}

		sm.handleRemoteBytes(streamID, data)
	}
}

func (sm *StringMux) AcceptConn() (Conn, error) {
	conn, ok := <-sm.connChan
	if !ok {
		return nil, fmt.Errorf("StringMux closed")
	}

	sm.logger.Printf("%s AcceptConn", conn)

	return conn, nil
}

func (sm *StringMux) GetConn(streamID string) (Conn, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	select {
	case <-sm.closeChan:
		return nil, fmt.Errorf("StringMux closed")
	default:
	}

	if _, ok := sm.conns[streamID]; ok {
		return nil, fmt.Errorf("Connection already exists")
	}

	c := sm.createConn(streamID, false)

	sm.logger.Printf("%s GetConn", c)

	return c, nil
}

func (sm *StringMux) handleRemoteBytes(streamID string, buf []byte) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	select {
	case <-sm.closeChan:
		// Check if StringMux was closed.
		sm.logger.Println("Ignoring remote data because connection has been closed")
		return
	default:
	}

	c := sm.getOrCreateConn(streamID, true)
	select {
	case <-c.closeChan:
		// Check if only the muxed connection was closed.
		sm.logger.Println("Ignoring remote data because connection has been closed")
		return
	default:
	}

	c.handleRemoteBytes(buf)
}

func (sm *StringMux) getOrCreateConn(streamID string, accept bool) *conn {
	c, ok := sm.conns[streamID]
	if !ok {
		c = sm.createConn(streamID, accept)
	}
	return c
}

func (sm *StringMux) Close() error {
	sm.logger.Println("StringMux Close")

	err := sm.params.Conn.Close()

	sm.close()

	sm.wg.Wait()

	return err
}

func (sm *StringMux) CloseChannel() <-chan struct{} {
	return sm.closeChan
}

func (sm *StringMux) close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, conn := range sm.conns {
		sm.closeConn(conn)
	}

	sm.closeOnce.Do(func() {
		close(sm.connChan)
		close(sm.closeChan)
	})
}

// handleClose calls closeConn ,but holds the lock.
func (sm *StringMux) handleClose(conn *conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.closeConn(conn)
}

// closeConn caller must hold the lock.
func (sm *StringMux) closeConn(conn *conn) {
	sm.logger.Printf("%s closeConn", conn)

	conn.onceClose.Do(func() {
		close(conn.readChan)
		close(conn.closeChan)
	})
	delete(sm.conns, conn.RemoteAddr().String())
}

func (sm *StringMux) createConn(streamID string, accept bool) *conn {
	c := &conn{
		streamID:    streamID,
		conn:        sm.params.Conn,
		readChan:    make(chan []byte, sm.params.ReadChanSize),
		closeChan:   make(chan struct{}),
		onClose:     sm.handleClose,
		debugLogger: sm.debugLogger,
	}
	sm.conns[streamID] = c
	if accept {
		sm.connChan <- c
	}
	return c
}

type Conn interface {
	net.Conn
	StreamID() string
	CloseChannel() <-chan struct{}
}

type conn struct {
	conn        net.Conn
	streamID    string
	readChan    chan []byte
	closeChan   chan struct{}
	onClose     func(*conn)
	onceClose   sync.Once
	debugLogger logger.Logger
}

var _ Conn = &conn{}

func (c *conn) StreamID() string {
	return c.streamID
}

func (c *conn) Close() error {
	c.onClose(c)
	return nil
}

func (c *conn) handleRemoteBytes(buf []byte) {
	b := make([]byte, len(buf))
	copy(b, buf)
	c.readChan <- b
}

func (c *conn) CloseChannel() <-chan struct{} {
	return c.closeChan
}

func (c *conn) Read(b []byte) (int, error) {
	buf, ok := <-c.readChan
	if !ok {
		return 0, fmt.Errorf("Conn closed")
	}
	copy(b, buf)
	c.debugLogger.Printf("%s recv %v", c, buf)
	return len(buf), nil
}

func (c *conn) Write(b []byte) (int, error) {
	select {
	case <-c.closeChan:
		return 0, fmt.Errorf("Conn is closed")
	default:
		data, err := Marshal(c.streamID, b)
		if err != nil {
			return 0, fmt.Errorf("Error marshalling data during write: %w", err)
		}
		c.debugLogger.Printf("%s send %v", c, b)
		_, err = c.conn.Write(data)
		if err != nil {
			return 0, fmt.Errorf("Error writing data: %w", err)
		}
		return len(b), nil
	}
}

func (c *conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *conn) String() string {
	if s, ok := c.conn.(stringer); ok {
		return fmt.Sprintf("%s [%s]", s.String(), c.streamID)
	}

	return fmt.Sprintf("[%s]", c.streamID)
}

type stringer interface {
	String() string
}
