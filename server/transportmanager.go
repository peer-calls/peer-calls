package server

import (
	"fmt"
	"net"
	"sync"
	"time"

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

func (t *TransportManager) Factories() []*ServerTransportFactory {
	t.mu.Lock()
	defer t.mu.Unlock()

	factories := make([]*ServerTransportFactory, 0, len(t.factories))

	for _, factory := range t.factories {
		factories = append(factories, factory)
	}

	return factories
}

func (t *TransportManager) start() {
	for {
		conn, err := t.udpMux.AcceptConn()
		if err != nil {
			t.logger.Printf("Error accepting udpMux conn: %s", err)
			return
		}

		t.logger.Printf("Accept UDP connection: %s", conn.RemoteAddr())

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
// provided connection.
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
	logger         Logger
	loggerFactory  LoggerFactory
	stringMux      *stringmux.StringMux
	transportsChan chan *StreamTransport
	transports     map[string]*StreamTransport
	promises       map[string]*TransportPromise
	mu             sync.Mutex
	wg             *sync.WaitGroup
}

func NewServerTransportFactory(
	loggerFactory LoggerFactory,
	wg *sync.WaitGroup,
	stringMux *stringmux.StringMux,
) *ServerTransportFactory {
	return &ServerTransportFactory{
		logger:         loggerFactory.GetLogger("stfactory"),
		loggerFactory:  loggerFactory,
		stringMux:      stringMux,
		transportsChan: make(chan *StreamTransport),
		transports:     map[string]*StreamTransport{},
		promises:       map[string]*TransportPromise{},
		wg:             wg,
	}
}

func (t *ServerTransportFactory) addPendingPromise(tp *TransportPromise) (ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	streamID := tp.StreamID()

	if _, ok := t.transports[streamID]; ok {
		return false
	}

	if _, ok := t.promises[streamID]; ok {
		return false
	}

	t.promises[streamID] = tp
	return true
}

func (t *ServerTransportFactory) removePendingPromiseWhenDone(tp *TransportPromise) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		_, _ = tp.Wait()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.promises, tp.StreamID())
	}()
}

// AcceptTransport returns a TransportPromise. This promise can be either
// canceled by using the Cancel method, or it can be Waited for by using the
// Wait method. The Wait() method must be called and the error must be checked
// and handled.
func (t *ServerTransportFactory) AcceptTransport() *TransportPromise {
	conn, err := t.stringMux.AcceptConn()

	if err != nil {
		tp := NewTransportPromise("", t.wg)
		tp.reject(fmt.Errorf("Error AcceptTransport: %w", err))
		return tp
	}

	tp := NewTransportPromise(conn.StreamID(), t.wg)

	if !t.addPendingPromise(tp) {
		tp.reject(fmt.Errorf("Promise or tranport already exists: %w", err))
		return tp
	}

	t.removePendingPromiseWhenDone(tp)

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

	// Ensure we don't get stuck at sctp.Client() or sctp.Server() forever.
	tp.onCancel(func() {
		tp.reject(ErrCanceled)
		_ = localMux.Close()
	})

	// TODO maybe we'll need to handle localMux Accept as well

	result, err := t.getOrAcceptStringMux(localMux, map[string]struct{}{
		"s": {},
		"m": {},
	})

	if err != nil {
		localMux.Close()
		tp.done(nil, fmt.Errorf("Error creating 's' and 'r' conns for raddr: %s %s: %w", raddr, streamID, err))
		return
	}

	sctpConn := result["s"]
	mediaConn := result["m"]

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

func (t *ServerTransportFactory) getOrAcceptStringMux(localMux *stringmux.StringMux, reqStreamIDs map[string]struct{}) (conns map[string]stringmux.Conn, errConn error) {
	var localMu sync.Mutex
	localWaitCh := make(chan struct{})
	localWaitChOnceClose := sync.Once{}

	conns = make(map[string]stringmux.Conn, len(reqStreamIDs))

	handleConn := func(conn stringmux.Conn) {
		localMu.Lock()
		defer localMu.Unlock()

		if _, ok := reqStreamIDs[conn.StreamID()]; ok {
			conns[conn.StreamID()] = conn
		} else {
			t.logger.Printf("%s Unexpected connection", conn)

			// // drain data from blocking the event loop

			// t.wg.Add(1)
			// go func() {
			// 	defer t.wg.Done()

			// 	buf := make([]byte, 1500)
			// 	for {
			// 		_, err := conn.Read(buf)
			// 		if err != nil {
			// 			return
			// 		}
			// 	}
			// }()
		}

		if len(reqStreamIDs) == len(conns) {
			localWaitChOnceClose.Do(func() {
				close(localWaitCh)
			})
		}
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		for {
			conn, err := localMux.AcceptConn()
			if err != nil {
				localWaitChOnceClose.Do(func() {
					// existing connections should be closed here so no need to close.
					errConn = err
					close(localWaitCh)
				})
				return
			}

			handleConn(conn)
		}
	}()

	for reqStreamID := range reqStreamIDs {
		if conn, err := localMux.GetConn(reqStreamID); err == nil {
			handleConn(conn)
		}
	}

	if len(reqStreamIDs) > 0 {
		<-localWaitCh
	}
	return
}

func (t *ServerTransportFactory) createTransport(
	raddr net.Addr,
	streamID string,
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

	t.mu.Lock()
	t.transports[streamID] = streamTransport
	t.mu.Unlock()

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

func (t *ServerTransportFactory) CloseTransport(streamID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if tp, ok := t.promises[streamID]; ok {
		// TODO check what happens when Cancel() is called later than resolve(). I
		// think this might still cause the transport to be created and added to
		// the transports map but not sure how to tackle this at this point.
		//
		// The good thing is that the promise will still be set by the time the
		// transport is added to transports map, but I'm still not 100% sure that
		// it will cover all edge cases.
		tp.Cancel()
	}

	if transport, ok := t.transports[streamID]; ok {
		transport.Close()
	}
}

// NewTransport returns a TransportPromise. This promise can be either canceled
// by using the Cancel method, or it can be Waited for by using the Wait
// method. The Wait() method must be called and the error must be checked and
// handled.
func (t *ServerTransportFactory) NewTransport(streamID string) *TransportPromise {
	tp := NewTransportPromise(streamID, t.wg)

	if !t.addPendingPromise(tp) {
		tp.reject(fmt.Errorf("Promise or transport already exists: %s", streamID))
		return tp
	}

	t.removePendingPromiseWhenDone(tp)

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
	streamID     string
	promise      promise.Promise
	cancelChan   chan struct{}
	transport    *StreamTransport
	resolveOnce  sync.Once
	cancelOnce   sync.Once
	onCancelOnce sync.Once
	onCancelHdlr func()
	wg           *sync.WaitGroup
}

func NewTransportPromise(streamID string, wg *sync.WaitGroup) *TransportPromise {
	return &TransportPromise{
		promise:    promise.New(),
		cancelChan: make(chan struct{}),
		transport:  nil,
		wg:         wg,
		streamID:   streamID,
	}
}

func (t *TransportPromise) StreamID() string {
	return t.streamID
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
			close(t.cancelChan)

			if t.onCancelHdlr != nil {
				t.onCancelHdlr()
			}

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
	err := t.promise.Wait()
	return t.transport, err
}

// WaitTimeout behaes similar to Wait, except it will automatically cancel the
// promise after a timeout.
func (t *TransportPromise) WaitTimeout(d time.Duration) (*StreamTransport, error) {
	select {
	case <-t.promise.WaitChannel():
	case <-time.After(d):
		t.Cancel()
	}
	return t.Wait()
}
