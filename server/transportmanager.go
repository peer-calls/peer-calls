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
	params        *TransportManagerParams
	udpMux        *udpmux.UDPMux
	closeChan     chan struct{}
	factoriesChan chan *ServerTransportFactory
	closeOnce     sync.Once
	mu            sync.Mutex
	wg            sync.WaitGroup
	logger        Logger

	factories map[*stringmux.StringMux]*ServerTransportFactory
}

type StreamTransport struct {
	Transport
	StreamID string

	association *sctp.Association
	stringMux   *stringmux.StringMux
}

func (st *StreamTransport) Close() error {
	err := st.Transport.Close()

	_ = st.association.Close()

	_ = st.stringMux.Close()

	return err
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
		params:        &params,
		udpMux:        udpMux,
		closeChan:     make(chan struct{}),
		factoriesChan: make(chan *ServerTransportFactory),
		factories:     make(map[*stringmux.StringMux]*ServerTransportFactory),
		logger:        params.LoggerFactory.GetLogger("transportmanager"),
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.start()
	}()

	return t
}

func (t *TransportManager) start() {
	for {
		conn, err := t.udpMux.AcceptConn()
		if err != nil {
			t.logger.Printf("Error accepting udpMux conn: %s", err)
			return
		}

		factory, err := t.createServerTransportFactory(conn)
		if err != nil {
			t.logger.Printf("Error creating transport factory: %s", err)
			return
		}

		t.factoriesChan <- factory
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

func (t *TransportManager) AcceptTransportFactory() (*ServerTransportFactory, error) {
	factory, ok := <-t.factoriesChan
	if !ok {
		return nil, fmt.Errorf("TransportManager is tearing down")
	}
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

func (t *TransportManager) CloseChannel() <-chan struct{} {
	return t.closeChan
}

func (t *TransportManager) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.udpMux.Close()

	t.closeOnce.Do(func() {
		close(t.factoriesChan)

		for stringMux, factory := range t.factories {
			_ = stringMux.Close()

			factory.Close()

			delete(t.factories, stringMux)
		}

		close(t.closeChan)
	})

	return err
}

type ServerTransportFactory struct {
	loggerFactory  LoggerFactory
	stringMux      *stringmux.StringMux
	transportsChan chan *StreamTransport
	transports     map[string]*StreamTransport
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
		transports:     map[string]*StreamTransport{},
		wg:             wg,
	}
}

// AcceptTransport returns a TransportPromise. This promise can be either
// canceled by using the Cancel method, or it can be Waited for by using the
// Wait method. The Wait() method must be called and the error must be checked
// and handled.
func (t *ServerTransportFactory) AcceptTransport() *TransportPromise {
	conn, err := t.stringMux.AcceptConn()

	tp := NewTransportPromise()

	if err != nil {
		tp.reject(fmt.Errorf("Error AcceptTransport: %w", err))
		return tp
	}

	t.createTransportAsync(tp, conn)

	return tp
}

func (t *ServerTransportFactory) createTransportAsync(tp *TransportPromise, conn stringmux.Conn) {
	go func() {
		transport, err := t.createTransport(conn)
		tp.done(transport, err)
	}()
}

func (t *ServerTransportFactory) createTransport(conn stringmux.Conn) (*StreamTransport, error) {
	streamID := conn.StreamID()

	stringMux2 := stringmux.New(stringmux.Params{
		Conn:          conn,
		LoggerFactory: t.loggerFactory,
		MTU:           uint32(receiveMTU),
		ReadChanSize:  100,
	})

	// TODO handle stringmux2.Accept

	sctpConn, err := stringMux2.GetConn("s")
	if err != nil {
		return nil, fmt.Errorf("Error creating 's' conn for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	mediaConn, err := stringMux2.GetConn("m")
	if err != nil {
		sctpConn.Close()
		return nil, fmt.Errorf("Error creating 'm' conn for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	// TODO this is a blocking method. figure out how to deal with this IRL

	association, err := sctp.Client(sctp.Config{
		NetConn:       sctpConn,
		LoggerFactory: NewPionLoggerFactory(t.loggerFactory),
		// MaxReceiveBufferSize: uint32(receiveMTU),
	})

	if err != nil {
		sctpConn.Close()
		mediaConn.Close()
		return nil, fmt.Errorf("Error creating sctp association for raddr: %s %s: %w", conn.RemoteAddr(), conn.StreamID(), err)
	}

	// TODO check if handling association.Accept is necessary since OpenStream
	// can return an error

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

	streamTransport := &StreamTransport{transport, streamID, association, stringMux2}
	t.transports[streamID] = streamTransport

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-transport.CloseChannel()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.transports, streamID)
	}()

	return streamTransport, nil
}

// NewTransport returns a TransportPromise. This promise can be either canceled
// by using the Cancel method, or it can be Waited for by using the Wait
// method. The Wait() method must be called and the error must be checked and
// handled.
func (t *ServerTransportFactory) NewTransport(streamID string) *TransportPromise {
	t.mu.Lock()
	defer t.mu.Unlock()

	tp := NewTransportPromise()

	if _, ok := t.transports[streamID]; ok {
		tp.reject(fmt.Errorf("Transport already exists: %s", streamID))
		return tp
	}

	conn, err := t.stringMux.GetConn(streamID)

	if err != nil {
		tp.reject(fmt.Errorf("Error retrieving transport conn: %s: %s", streamID, err))
		return tp
	}

	t.createTransportAsync(tp, conn)

	return tp
}

func (t *ServerTransportFactory) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for streamID, transport := range t.transports {
		transport.Close()
		delete(t.transports, streamID)
	}

	return nil
}

type TransportPromise struct {
	promise    promise.Promise
	cancelChan chan struct{}
	transport  *StreamTransport
	once       sync.Once
	onceCancel sync.Once
}

func NewTransportPromise() *TransportPromise {
	return &TransportPromise{
		promise:   promise.New(),
		transport: nil,
	}
}

func (t *TransportPromise) done(transport *StreamTransport, err error) {
	t.once.Do(func() {
		if err != nil {
			t.promise.Reject(err)
		} else {
			t.transport = transport
			t.promise.Resolve()
		}
	})
}

func (t *TransportPromise) resolve(transport *StreamTransport) {
	t.done(transport, nil)
}

func (t *TransportPromise) reject(err error) {
	t.done(nil, err)
}

var ErrCanceled = fmt.Errorf("Canceled")

// Cancel waits for the transport in another goroutine and closes it as soon as
// the promise resolves.
func (t *TransportPromise) Cancel() {
	go func() {
		t.onceCancel.Do(func() {
			close(t.cancelChan)

			_ = t.promise.Wait()
			if t.transport != nil {
				t.transport.Close()
			}
		})
	}()
}

// Wait returns the Transport or error after the promise is resolved or
// rejected. Promise can be rejected if an error occurs or if a promise is
// canceled using the Cancel function.
func (t *TransportPromise) Wait() (*StreamTransport, error) {
	select {
	case <-t.promise.WaitChannel():
		err := t.promise.Wait()
		return t.transport, err
	case <-t.cancelChan:
		return nil, ErrCanceled
	}
}
