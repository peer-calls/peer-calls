package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/peer-calls/peer-calls/server/promise"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/udpmux"
	"github.com/pion/sctp"
)

// TransportManager is in charge of managing server-to-server transports.
type TransportManager struct {
	params    *TransportManagerParams
	udpMux    *udpmux.UDPMux
	connCh    chan *ServerTransportFactory
	closeOnce sync.Once
	mu        sync.Mutex
	wg        sync.WaitGroup
	logger    Logger

	factories map[*stringmux.StringMux]*ServerTransportFactory
}

type StreamTransport struct {
	Transport
	StreamID string
}

type TransportManagerParams struct {
	Conn          net.PacketConn
	LoggerFactory LoggerFactory
}

func NewTransportManager(params TransportManagerParams) *TransportManager {
	udpMux := udpmux.New(udpmux.Params{
		Conn:          params.Conn,
		MTU:           uint32(receiveMTU),
		LoggerFactory: params.LoggerFactory,
		ReadChanSize:  100,
	})

	t := &TransportManager{
		params:    &params,
		udpMux:    udpMux,
		factories: make(map[*stringmux.StringMux]*ServerTransportFactory),
	}

	go t.start()

	return t
}

func (t *TransportManager) start() {
	for {
		conn, err := t.udpMux.AcceptConn()
		if err != nil {
			t.logger.Printf("Error accepting udpMux conn: %s", err)
			return
		}

		t.createServerTransportFactory(conn)
	}
}

func (t *TransportManager) create(conn udpmux.Conn) {
	t.createServerTransportFactory(conn)
}

// createServerTransportFactory creates a new ServerTransportFactory for the
// provided association.
func (t *TransportManager) createServerTransportFactory(conn udpmux.Conn) (*ServerTransportFactory, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	stringMux := stringmux.New(stringmux.Params{
		LoggerFactory: t.params.LoggerFactory,
		Conn:          conn,
		MTU:           uint32(receiveMTU), // TODO not sure if this is ok
		ReadChanSize:  100,
	})

	factory := NewServerTransportFactory(t.params.LoggerFactory, &t.wg, stringMux)
	t.factories[stringMux] = factory

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-stringMux.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.factories, stringMux)
	}()

	return factory, nil
}

func (t *TransportManager) GetTransportFactory(raddr net.Addr) (*ServerTransportFactory, error) {
	conn, err := t.udpMux.GetConn(raddr)
	if err != nil {
		return nil, fmt.Errorf("Error getting conn for raddr %s: %w", raddr, err)
	}

	return t.createServerTransportFactory(conn)
}

func (t *TransportManager) Close() error {
	err := t.close()

	t.wg.Wait()

	return err
}

func (t *TransportManager) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.udpMux.Close()

	t.closeOnce.Do(func() {
		for stringMux := range t.factories {
			_ = stringMux.Close()
			delete(t.factories, stringMux)
		}
	})

	return err
}

type ServerTransportFactory struct {
	loggerFactory  LoggerFactory
	stringMux      *stringmux.StringMux
	transportsChan chan *StreamTransport
	Transports     map[string]*StreamTransport
	mu             sync.Mutex
	wg             *sync.WaitGroup
}

func NewServerTransportFactory(
	loggerFactory LoggerFactory,
	wg *sync.WaitGroup,
	stringMux *stringmux.StringMux,
) *ServerTransportFactory {
	return &ServerTransportFactory{
		loggerFactory:  loggerFactory,
		stringMux:      stringMux,
		transportsChan: make(chan *StreamTransport),
		Transports:     map[string]*StreamTransport{},
		wg:             wg,
	}
}

func (t *ServerTransportFactory) AcceptTransport() (*StreamTransport, error) {
	conn, err := t.stringMux.AcceptConn()

	if err != nil {
		return nil, fmt.Errorf("Error AcceptTransport: %w", err)
	}

	return t.createTransport(conn)
}

func (t *ServerTransportFactory) createTransport(conn stringmux.Conn) (*StreamTransport, error) {
	streamID := conn.StreamID()

	stringmux2 := stringmux.New(stringmux.Params{
		Conn:          conn,
		LoggerFactory: t.loggerFactory,
		MTU:           uint32(receiveMTU),
		ReadChanSize:  100,
	})

	// TODO handle stringmux2.Accept

	sctpConn, err := stringmux2.GetConn("s")
	if err != nil {
		return nil, fmt.Errorf("Error creating 's' conn for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	mediaConn, err := stringmux2.GetConn("m")
	if err != nil {
		sctpConn.Close()
		return nil, fmt.Errorf("Error creating 'm' conn for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	// TODO this is a blocking method. figure out how to deal with this IRL

	association, err := sctp.Client(sctp.Config{
		NetConn:              sctpConn,
		LoggerFactory:        NewPionLoggerFactory(t.loggerFactory),
		MaxReceiveBufferSize: 100,
	})

	if err != nil {
		sctpConn.Close()
		mediaConn.Close()
		return nil, fmt.Errorf("Error creating sctp association for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	// TODO handle association.Accept

	// TODO figure out what to do when we get an error that a stream already exists. wait for Accept?
	metadataStream, err := association.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		sctpConn.Close()
		mediaConn.Close()
		association.Close()
		return nil, fmt.Errorf("Error creating metadata sctp stream for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	dataStream, err := association.OpenStream(1, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		sctpConn.Close()
		mediaConn.Close()
		association.Close()
		metadataStream.Close()
		return nil, fmt.Errorf("Error creating data sctp stream for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	transport := NewServerTransport(t.loggerFactory, mediaConn, dataStream, metadataStream)

	streamTransport := &StreamTransport{transport, streamID}
	t.Transports[streamID] = streamTransport

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-transport.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.Transports, streamID)
	}()

	return streamTransport, nil
}

func (t *ServerTransportFactory) NewTransport(streamID string) (*StreamTransport, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.Transports[streamID]; ok {
		return nil, fmt.Errorf("Transport already exists: %s", streamID)
	}

	conn, err := t.stringMux.GetConn(streamID)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving stringmux conn: %w", err)
	}

	return t.createTransport(conn)
}

type TransportPromise struct {
	promise    promise.Promise
	transport  Transport
	once       sync.Once
	onceCancel sync.Once
}

func NewTransportPromise() *TransportPromise {
	return &TransportPromise{
		promise:   promise.New(),
		transport: nil,
	}
}

func (t *TransportPromise) done(transport Transport, err error) {
	t.once.Do(func() {
		if err != nil {
			t.promise.Reject(err)
		} else {
			t.transport = transport
			t.promise.Resolve()
		}
	})
}

func (t *TransportPromise) resolve(transport Transport) {
	t.done(transport, nil)
}

func (t *TransportPromise) reject(err error) {
	t.done(nil, err)
}

var ErrCanceled = fmt.Errorf("Canceled")

func (t *TransportPromise) Cancel() {
	go func() {
		t.onceCancel.Do(func() {
			t.promise.Wait()
			if t.transport != nil {
				t.transport.Close()
			}
		})
	}()
}

func (t *TransportPromise) Wait() (Transport, error) {
	err := t.promise.Wait()
	return t.transport, err
}
