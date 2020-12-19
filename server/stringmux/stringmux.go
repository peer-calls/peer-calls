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

	logger      logger.Logger
	debugLogger logger.Logger

	getConnRequestChan   chan getConnRequest
	newConnChan          chan Conn
	closeConnRequestChan chan closeConnRequest
	remotePacketsChan    chan remotePacket

	teardownChan chan struct{}
	torndownChan chan struct{}
}

type Params struct {
	LoggerFactory  logger.LoggerFactory
	Conn           net.Conn
	MTU            uint32
	ReadChanSize   int
	ReadBufferSize int
}

func New(params Params) *StringMux {
	m := &StringMux{
		params: &params,

		logger:      params.LoggerFactory.GetLogger("stringmux:info"),
		debugLogger: params.LoggerFactory.GetLogger("stringmux:debug"),

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
			debugLogger: m.debugLogger,

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
			m.debugLogger.Printf("dropped packet for conn: %s, streamID: %s", conn, streamID)
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
			m.logger.Printf("Error reading remote data: %s", err)

			return
		}

		streamID, data, err := Unmarshal(buf[:i])
		if err != nil {
			m.logger.Printf("Error unmarshaling remote data: %s", err)

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

	m.logger.Printf("%s AcceptConn", conn)

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
