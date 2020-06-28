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

	tp := NewTransportPromise(t.wg)

	if err != nil {
		tp.reject(fmt.Errorf("Error AcceptTransport: %w", err))
		return tp
	}

	t.createTransportAsync(tp, conn, true)

	return tp
}

func (t *ServerTransportFactory) createTransportAsync(tp *TransportPromise, conn stringmux.Conn, server bool) {
	raddr := conn.RemoteAddr()
	streamID := conn.StreamID()

	// This can be optimized in the future since a StringMux has a minimal
	// overhead of 3 bytes, and only a single bit is needed.
	localMux := stringmux.New(stringmux.Params{
		Conn:          conn,
		LoggerFactory: t.loggerFactory,
		MTU:           uint32(receiveMTU),
		ReadChanSize:  100,
	})

	// TODO maybe we'll need to handle localMux Accept as well

	sctpConn, err := localMux.GetConn("s")
	if err != nil {
		localMux.Close()
		tp.done(nil, fmt.Errorf("Error creating 's' conn for raddr: %s %s: %w", raddr, streamID, err))
		return
	}

	mediaConn, err := localMux.GetConn("m")
	if err != nil {
		sctpConn.Close()
		localMux.Close()
		tp.done(nil, fmt.Errorf("Error creating 'm' conn for raddr: %s %s: %w", raddr, streamID, err))
		return
	}

	// Ensure we don't get stuck at sctp.Client() forever.
	tp.onCancel(func() {
		_ = localMux.Close()
	})

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		transport, err := t.createTransport(conn.RemoteAddr(), conn.StreamID(), localMux, mediaConn, sctpConn, server)
		if err != nil {
			mediaConn.Close()
			sctpConn.Close()
			localMux.Close()
		}

		tp.done(transport, err)
	}()
}

func (t *ServerTransportFactory) createTransport(
	raddr net.Addr, streamID string,
	localMux *stringmux.StringMux,
	mediaConn net.Conn,
	sctpConn net.Conn,
	server bool,
) (*StreamTransport, error) {
	sctpConfig := sctp.Config{
		NetConn:       sctpConn,
		LoggerFactory: NewPionLoggerFactory(t.loggerFactory),
	}

	var association *sctp.Association
	var err error

	if server {
		association, err = sctp.Server(sctpConfig)
	} else {
		association, err = sctp.Client(sctpConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("Error creating sctp association for raddr: %s %s: %w", raddr, streamID, err)
	}

	// TODO check if handling association.Accept is necessary since OpenStream
	// can return an error. Perhaps we need to wait for Accept as well, check the
	// StreamIdentifier and log stream IDs we are not expecting.

	metadataStream, err := association.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		association.Close()
		return nil, fmt.Errorf("Error creating metadata sctp stream for raddr: %s %s: %w", raddr, streamID, err)
	}

	dataStream, err := association.OpenStream(1, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		metadataStream.Close()
		association.Close()
		return nil, fmt.Errorf("Error creating data sctp stream for raddr: %s %s: %w", raddr, streamID, err)
	}

	transport := NewServerTransport(t.loggerFactory, mediaConn, dataStream, metadataStream)

	streamTransport := &StreamTransport{transport, streamID, association, localMux}
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

	tp := NewTransportPromise(t.wg)

	if _, ok := t.transports[streamID]; ok {
		tp.reject(fmt.Errorf("Transport already exists: %s", streamID))
		return tp
	}

	conn, err := t.stringMux.GetConn(streamID)

	if err != nil {
		tp.reject(fmt.Errorf("Error retrieving transport conn: %s: %s", streamID, err))
		return tp
	}

	t.createTransportAsync(tp, conn, false)

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
	promise      promise.Promise
	cancelChan   chan struct{}
	transport    *StreamTransport
	resolveOnce  sync.Once
	cancelOnce   sync.Once
	onCancelOnce sync.Once
	onCancelHdlr func()
	wg           *sync.WaitGroup
}

func NewTransportPromise(wg *sync.WaitGroup) *TransportPromise {
	return &TransportPromise{
		promise:    promise.New(),
		cancelChan: make(chan struct{}),
		transport:  nil,
		wg:         wg,
	}
}

func (t *TransportPromise) done(transport *StreamTransport, err error) {
	t.resolveOnce.Do(func() {
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

func (t *TransportPromise) onCancel(handleClose func()) {
	t.onCancelOnce.Do(func() {
		t.onCancelHdlr = handleClose
	})
}

var ErrCanceled = fmt.Errorf("Canceled")

// Cancel waits for the transport in another goroutine and closes it as soon as
// the promise resolves.
func (t *TransportPromise) Cancel() {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		t.cancelOnce.Do(func() {
			if t.onCancelHdlr != nil {
				t.onCancelHdlr()
			}

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
