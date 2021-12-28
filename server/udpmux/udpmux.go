package udpmux

import (
	"context"
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

const DefaultMTU uint32 = 8192

type UDPMux struct {
	params *Params

	getConnRequestChan   chan getConnRequest
	newConnChan          chan Conn
	closeConnRequestChan chan closeConnRequest
	remotePacketsChan    chan remotePacket

	teardownChan chan struct{}
	torndownChan chan struct{}
}

type Params struct {
	Conn           net.PacketConn
	MTU            uint32
	Log            logger.Logger
	ReadChanSize   int
	ReadBufferSize int
}

func New(params Params) *UDPMux {
	params.Log = params.Log.WithNamespaceAppended("udpmux").WithCtx(logger.Ctx{
		"local_addr": params.Conn.LocalAddr(),
	})

	m := &UDPMux{
		params: &params,

		newConnChan:          make(chan Conn),
		closeConnRequestChan: make(chan closeConnRequest),
		getConnRequestChan:   make(chan getConnRequest),
		remotePacketsChan:    make(chan remotePacket, params.ReadBufferSize),
		teardownChan:         make(chan struct{}, 1),
		torndownChan:         make(chan struct{}),
	}

	if m.params.MTU == 0 {
		m.params.MTU = DefaultMTU
	}

	go m.startLoop()

	return m
}

func (m *UDPMux) LocalAddr() net.Addr {
	return m.params.Conn.LocalAddr()
}

// Conns is the channel with incoming connections. Users should use either
// AcceptConn or Conns, but never both.
func (m *UDPMux) Conns() <-chan Conn {
	return m.newConnChan
}

// AcceptConn reads from Conns channel. It returns io.ErrClosedPipe when
// the channel is closed. Users should use either AcceptConn or Conns, but
// never both.
func (m *UDPMux) AcceptConn() (Conn, error) {
	c, ok := <-m.newConnChan
	if !ok {
		return nil, errors.Annotate(io.ErrClosedPipe, "accept")
	}

	c.(*conn).logger.Info("Accept conn", nil)

	return c, nil
}

func (m *UDPMux) GetConn(raddr net.Addr) (Conn, error) {
	req := getConnRequest{
		raddr:    raddr,
		connChan: make(chan Conn, 1),
		errChan:  make(chan error, 1),
	}

	select {
	case m.getConnRequestChan <- req:
	case <-m.torndownChan:
		return nil, errors.Annotatef(io.ErrClosedPipe, "get conn")
	}

	select {
	case err := <-req.errChan:
		return nil, errors.Trace(err)
	case conn := <-req.connChan:
		return conn, nil
	}
}

func (m *UDPMux) startLoop() {
	readCtx, readCancel := context.WithCancel(context.Background())
	readingDone := make(chan struct{})

	go func() {
		defer close(readingDone)
		m.startReading(readCtx)
	}()

	conns := map[string]*conn{}

	defer func() {
		_ = m.params.Conn.Close()

		readCancel()
		<-readingDone

		for _, conn := range conns {
			conn.close()
		}

		close(m.newConnChan)
		close(m.torndownChan)
	}()

	createConn := func(raddr net.Addr) *conn {
		return &conn{
			logger: m.params.Log.WithNamespaceAppended("conn").WithCtx(logger.Ctx{
				"remote_addr": raddr,
			}),

			conn:  m.params.Conn,
			laddr: m.params.Conn.LocalAddr(),
			raddr: raddr,

			readChan:             make(chan []byte, m.params.ReadChanSize),
			closeConnRequestChan: m.closeConnRequestChan,
			torndown:             make(chan struct{}),
		}
	}

	getNewConn := func(req getConnRequest) {
		raddrStr := req.raddr.String()

		_, ok := conns[raddrStr]
		if ok {
			req.errChan <- errors.Annotatef(ErrConnAlreadyExists, "raddr: %s", raddrStr)

			return
		}

		conn := createConn(req.raddr)
		conns[raddrStr] = conn

		req.connChan <- conn
	}

	acceptOrGet := func(conn *conn) bool {
		raddrStr := conn.RemoteAddr().String()

		for {
			select {
			case m.newConnChan <- conn:
				conn.logger.Debug("Accepted remote conn", nil)

				conns[raddrStr] = conn

				return true
			case req := <-m.getConnRequestChan:
				if req.raddr.String() != raddrStr {
					// Handle request for another connection.
					getNewConn(req)

					// But retry to advertise this conn.
					continue
				}

				req.connChan <- conn

				conns[raddrStr] = conn

				return true
			case <-m.teardownChan:
				return false
			}
		}
	}

	handlePacket := func(pkt remotePacket) bool {
		raddrStr := pkt.raddr.String()

		conn, ok := conns[raddrStr]
		if !ok {
			conn = createConn(pkt.raddr)

			if !acceptOrGet(conn) {
				return false
			}
		}

		select {
		case conn.readChan <- pkt.bytes:
		case <-conn.torndown:
		case <-m.teardownChan:
			return false
		}

		return true
	}

	handleClose := func(req closeConnRequest) {
		raddrStr := req.conn.raddr.String()

		conn, ok := conns[raddrStr]
		if !ok {
			req.errChan <- errors.Annotatef(ErrConnNotFound, "raddr: %s", raddrStr)

			return
		}

		if conn == req.conn {
			delete(conns, raddrStr)
			conn.close()
		}

		req.errChan <- nil
	}

	for {
		select {
		case req := <-m.getConnRequestChan:
			getNewConn(req)
		case pkt, ok := <-m.remotePacketsChan:
			if !ok {
				return
			}

			if ok := handlePacket(pkt); !ok {
				return
			}
		case req := <-m.closeConnRequestChan:
			handleClose(req)
		case <-m.teardownChan:
			return
		}
	}
}

func (m *UDPMux) startReading(ctx context.Context) {
	buf := make([]byte, m.params.MTU)
	done := ctx.Done()

	defer close(m.remotePacketsChan)

	for {
		i, raddr, err := m.params.Conn.ReadFrom(buf)
		if err != nil {
			m.params.Log.Error("read remote data", errors.Trace(err), nil)

			return
		}

		pkt := remotePacket{
			bytes: make([]byte, i),
			raddr: raddr,
		}

		copy(pkt.bytes, buf[:i])

		select {
		case m.remotePacketsChan <- pkt:
			// OK
		case <-done:
			return
		}
	}
}

func (m *UDPMux) Done() <-chan struct{} {
	return m.torndownChan
}

func (m *UDPMux) Close() error {
	select {
	case m.teardownChan <- struct{}{}:
	case <-m.torndownChan:
	}

	for range m.newConnChan {
		// Empty the newConnChan in case there is a new connection blocking on send.
	}

	<-m.torndownChan

	return nil
}

type remotePacket struct {
	bytes []byte
	raddr net.Addr
}

type getConnRequest struct {
	raddr net.Addr

	connChan chan Conn
	errChan  chan error
}

type closeConnRequest struct {
	conn    *conn
	errChan chan error
}
