package stringmux

import (
	"context"
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
)

const DefaultMTU uint32 = 8192

type StringMux struct {
	params *Params

	getConnRequestChan   chan getConnRequest
	newConnChan          chan *conn
	closeConnRequestChan chan closeConnRequest
	remotePacketsChan    chan remotePacket

	teardownChan chan struct{}
	torndownChan chan struct{}
}

type Params struct {
	Log            logger.Logger
	Conn           net.Conn
	MTU            uint32
	ReadChanSize   int
	ReadBufferSize int
}

func New(params Params) *StringMux {
	params.Log = params.Log.WithNamespaceAppended("stringmux").WithCtx(logger.Ctx{
		"local_addr":  params.Conn.LocalAddr(),
		"remote_addr": params.Conn.RemoteAddr(),
	})

	m := &StringMux{
		params: &params,

		newConnChan:          make(chan *conn),
		closeConnRequestChan: make(chan closeConnRequest),
		getConnRequestChan:   make(chan getConnRequest),
		remotePacketsChan:    make(chan remotePacket, params.ReadBufferSize),
		teardownChan:         make(chan struct{}, 1),
		torndownChan:         make(chan struct{}),
	}

	if m.params.MTU == 0 {
		m.params.MTU = DefaultMTU
	}

	go m.start()

	return m
}

func (m *StringMux) start() {
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

	createConn := func(streamID string) *conn {
		return &conn{
			logger: m.params.Log.WithNamespaceAppended("conn").WithCtx(logger.Ctx{
				"stream_id": streamID,
			}),

			conn:     m.params.Conn,
			streamID: streamID,

			readChan:             make(chan []byte, m.params.ReadChanSize),
			closeConnRequestChan: m.closeConnRequestChan,
			torndown:             make(chan struct{}),
		}
	}

	getNewConn := func(req getConnRequest) {
		streamID := req.streamID

		_, ok := conns[streamID]
		if ok {
			req.errChan <- errors.Annotatef(ErrConnAlreadyExists, "streamID: %s", streamID)

			return
		}

		conn := createConn(streamID)
		conns[streamID] = conn

		req.connChan <- conn
	}

	handlePacket := func(pkt remotePacket) {
		streamID := pkt.streamID

		conn, ok := conns[streamID]
		if !ok {
			conn = createConn(streamID)
			conns[streamID] = conn
			m.newConnChan <- conn
		}

		select {
		case conn.readChan <- pkt.bytes:
		default:
			conn.logger.Warn("Dropped packet", nil)
		}
	}

	handleClose := func(req closeConnRequest) {
		streamID := req.conn.streamID

		conn, ok := conns[streamID]
		if !ok {
			req.errChan <- errors.Annotatef(ErrConnNotFound, "streamID: %s", streamID)

			return
		}

		if conn == req.conn {
			delete(conns, streamID)
			conn.close()
		}

		req.errChan <- nil
	}

	for {
		select {
		case req := <-m.getConnRequestChan:
			getNewConn(req)
		case pkt := <-m.remotePacketsChan:
			handlePacket(pkt)
		case req := <-m.closeConnRequestChan:
			handleClose(req)
		case <-m.teardownChan:
			return
		}
	}
}

func (m *StringMux) startReading(ctx context.Context) {
	buf := make([]byte, m.params.MTU)
	done := ctx.Done()

	for {
		i, err := m.params.Conn.Read(buf)
		if err != nil {
			m.params.Log.Error("read remote data", errors.Trace(err), nil)

			return
		}

		streamID, data, err := Unmarshal(buf[:i])
		if err != nil {
			m.params.Log.Error("unmarshal remote data", errors.Trace(err), nil)

			return
		}

		pkt := remotePacket{
			bytes:    make([]byte, len(data)),
			streamID: streamID,
		}

		copy(pkt.bytes, data)

		select {
		case m.remotePacketsChan <- pkt:
			// OK
		case <-done:
			return
		}
	}
}

func (m *StringMux) AcceptConn() (Conn, error) {
	conn, ok := <-m.newConnChan
	if !ok {
		return nil, errors.Annotate(io.ErrClosedPipe, "accept")
	}

	conn.logger.Info("Accept conn", nil)

	return conn, nil
}

func (m *StringMux) GetConn(streamID string) (Conn, error) {
	req := getConnRequest{
		streamID: streamID,
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

func (m *StringMux) Close() error {
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

func (m *StringMux) CloseChannel() <-chan struct{} {
	return m.torndownChan
}

type remotePacket struct {
	bytes    []byte
	streamID string
}

type getConnRequest struct {
	streamID string

	connChan chan Conn
	errChan  chan error
}

type closeConnRequest struct {
	conn    *conn
	errChan chan error
}
