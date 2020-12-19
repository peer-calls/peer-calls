package server

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/udpmux"
	"github.com/pion/sctp"
)

// TransportManager is in charge of managing server-to-server transports.
type TransportManager struct {
	params        *TransportManagerParams
	udpMux        *udpmux.UDPMux
	closeChan     chan struct{}
	factoriesChan chan *TransportFactory
	closeOnce     sync.Once
	mu            sync.Mutex
	wg            sync.WaitGroup
	logger        Logger

	factories map[*stringmux.StringMux]*TransportFactory
}

type StreamTransport struct {
	Transport
	StreamID string

	association *sctp.Association
	stringMux   *stringmux.StringMux
}

func (st *StreamTransport) Close() error {
	var errs MultiErrorHandler

	errs.Add(errors.Annotate(st.Transport.Close(), "close transport"))
	errs.Add(errors.Annotate(st.association.Close(), "close association"))
	errs.Add(errors.Annotate(st.stringMux.Close(), "close string mux"))

	return errors.Annotate(errs.Err(), "close stream transport")
}

type TransportManagerParams struct {
	Conn          net.PacketConn
	LoggerFactory LoggerFactory
}

func NewTransportManager(params TransportManagerParams) *TransportManager {
	udpMux := udpmux.New(udpmux.Params{
		Conn:           params.Conn,
		MTU:            uint32(receiveMTU),
		LoggerFactory:  params.LoggerFactory,
		ReadChanSize:   100,
		ReadBufferSize: 0,
	})

	t := &TransportManager{
		params:        &params,
		udpMux:        udpMux,
		closeChan:     make(chan struct{}),
		factoriesChan: make(chan *TransportFactory),
		factories:     make(map[*stringmux.StringMux]*TransportFactory),
		logger:        params.LoggerFactory.GetLogger("transportmanager"),
	}

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		t.start()
	}()

	return t
}

func (t *TransportManager) Factories() []*TransportFactory {
	t.mu.Lock()
	defer t.mu.Unlock()

	factories := make([]*TransportFactory, 0, len(t.factories))

	for _, factory := range t.factories {
		factories = append(factories, factory)
	}

	return factories
}

func (t *TransportManager) start() {
	for {
		conn, err := t.udpMux.AcceptConn()
		if err != nil {
			t.logger.Printf("Error accepting udpMux conn: %+v", err)
			return
		}

		t.logger.Printf("Accept UDP connection: %s", conn.RemoteAddr())

		factory, err := t.createTransportFactory(conn)
		if err != nil {
			t.logger.Printf("Error creating transport factory: %+v", err)
			return
		}

		t.factoriesChan <- factory
	}
}

// createTransportFactory creates a new TransportFactory for the
// provided connection.
func (t *TransportManager) createTransportFactory(conn udpmux.Conn) (*TransportFactory, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	stringMux := stringmux.New(stringmux.Params{
		LoggerFactory:  t.params.LoggerFactory,
		Conn:           conn,
		MTU:            uint32(receiveMTU), // TODO not sure if this is ok
		ReadChanSize:   100,
		ReadBufferSize: 0,
	})

	factory := NewTransportFactory(t.params.LoggerFactory, &t.wg, stringMux)
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

func (t *TransportManager) AcceptTransportFactory() (*TransportFactory, error) {
	factory, ok := <-t.factoriesChan
	if !ok {
		return nil, errors.Annotate(io.ErrClosedPipe, "TransportManager is tearing down")
	}
	return factory, nil
}

func (t *TransportManager) GetTransportFactory(raddr net.Addr) (*TransportFactory, error) {
	conn, err := t.udpMux.GetConn(raddr)
	if err != nil {
		return nil, errors.Annotatef(err, "getting conn for raddr: %s", raddr)
	}

	return t.createTransportFactory(conn)
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

type TransportFactory struct {
	logger            Logger
	loggerFactory     LoggerFactory
	stringMux         *stringmux.StringMux
	transportsChan    chan *StreamTransport
	transports        map[string]*StreamTransport
	pendingTransports map[string]*TransportRequest
	mu                sync.Mutex
	wg                *sync.WaitGroup
}

func NewTransportFactory(
	loggerFactory LoggerFactory,
	wg *sync.WaitGroup,
	stringMux *stringmux.StringMux,
) *TransportFactory {
	return &TransportFactory{
		logger:            loggerFactory.GetLogger("stfactory"),
		loggerFactory:     loggerFactory,
		stringMux:         stringMux,
		transportsChan:    make(chan *StreamTransport),
		transports:        map[string]*StreamTransport{},
		pendingTransports: map[string]*TransportRequest{},
		wg:                wg,
	}
}

func (t *TransportFactory) addPendingTransport(req *TransportRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	streamID := req.StreamID()

	if _, ok := t.transports[streamID]; ok {
		return errors.Errorf("transport already exist: %s", streamID)
	}

	if _, ok := t.pendingTransports[streamID]; ok {
		return errors.Errorf("transport promise already exists: %s", streamID)
	}

	t.pendingTransports[streamID] = req
	return nil
}

func (t *TransportFactory) removePendingPromiseWhenDone(req *TransportRequest) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		<-req.Done()

		t.mu.Lock()
		defer t.mu.Unlock()

		delete(t.pendingTransports, req.StreamID())
	}()
}

// AcceptTransport returns a TransportRequest. This promise can be either
// canceled by using the Cancel method, or it can be Waited for by using the
// Wait method. The Wait() method must be called and the error must be checked
// and handled.
func (t *TransportFactory) AcceptTransport() *TransportRequest {
	conn, err := t.stringMux.AcceptConn()

	if err != nil {
		req := NewTransportRequest(context.Background(), "")
		req.set(nil, errors.Annotate(err, "accept transport"))
		return req
	}

	streamID := conn.StreamID()

	req := NewTransportRequest(context.Background(), streamID)

	if err := t.addPendingTransport(req); err != nil {
		req.set(nil, errors.Annotatef(err, "accept: promise or transport already exists: %s", streamID))
		return req
	}

	t.removePendingPromiseWhenDone(req)

	t.createTransportAsync(req, conn, true)

	return req
}

func (t *TransportFactory) createTransportAsync(req *TransportRequest, conn stringmux.Conn, server bool) {
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

	// transportCreated will be closed as soon as the goroutine from which
	// createTransport is called is done.
	transportCreated := make(chan struct{})

	t.wg.Add(1)

	// The following gouroutine waits for the request context to be done
	// (canceled) and closes the local mux so that the goroutine from which
	// createTransport is called does not block forever.
	go func() {
		defer t.wg.Done()

		select {
		case <-req.Context().Done():
			// Ensure we don't get stuck at sctp.Client() or sctp.Server() forever.
			_ = localMux.Close()
		case <-transportCreated:
		}
	}()

	// TODO maybe we'll need to handle localMux Accept as well

	result, err := t.getOrAcceptStringMux(localMux, map[string]struct{}{
		"s": {},
		"m": {},
	})

	if err != nil {
		localMux.Close()
		req.set(nil, errors.Annotatef(err, "creating 's' and 'r' conns for raddr: %s %s", raddr, streamID))
		return
	}

	sctpConn := result["s"]
	mediaConn := result["m"]

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer close(transportCreated)

		transport, err := t.createTransport(conn.RemoteAddr(), conn.StreamID(), localMux, mediaConn, sctpConn, server)
		if err != nil {
			mediaConn.Close()
			sctpConn.Close()
			localMux.Close()
		}

		req.set(transport, errors.Trace(err))
	}()
}

func (t *TransportFactory) getOrAcceptStringMux(localMux *stringmux.StringMux, reqStreamIDs map[string]struct{}) (conns map[string]stringmux.Conn, errConn error) {
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

func (t *TransportFactory) createTransport(
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

	// if server {
	// 	association, err = sctp.Server(sctpConfig)
	// } else {
	association, err = sctp.Client(sctpConfig)
	// }

	if err != nil {
		return nil, errors.Annotatef(err, "creating sctp association for raddr: %s %s", raddr, streamID)
	}

	// TODO check if handling association.Accept is necessary since OpenStream
	// can return an error. Perhaps we need to wait for Accept as well, check the
	// StreamIdentifier and log stream IDs we are not expecting.

	metadataStream, err := association.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		association.Close()
		return nil, errors.Annotatef(err, "creating metadata sctp stream for raddr: %s %s", raddr, streamID)
	}

	dataStream, err := association.OpenStream(1, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		metadataStream.Close()
		association.Close()
		return nil, errors.Annotatef(err, "creating data sctp stream for raddr: %s %s", raddr, streamID)
	}

	transport := NewServerTransport(t.loggerFactory, mediaConn, dataStream, metadataStream)

	streamTransport := &StreamTransport{
		Transport:   transport,
		StreamID:    streamID,
		association: association,
		stringMux:   localMux,
	}

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

func (t *TransportFactory) CloseTransport(streamID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if tp, ok := t.pendingTransports[streamID]; ok {
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
		if err := transport.Close(); err != nil {
			t.logger.Printf("Error closing transport: %s: %+v", streamID, err)
		}
	}
}

// NewTransport returns a TransportRequest. This promise can be either canceled
// by using the Cancel method, or it can be Waited for by using the Wait
// method. The Wait() method must be called and the error must be checked and
// handled.
func (t *TransportFactory) NewTransport(streamID string) *TransportRequest {
	req := NewTransportRequest(context.Background(), streamID)

	if err := t.addPendingTransport(req); err != nil {
		req.set(nil, errors.Annotatef(err, "new: promise or transport already exists: %s", streamID))
		return req
	}

	t.removePendingPromiseWhenDone(req)

	conn, err := t.stringMux.GetConn(streamID)
	if err != nil {
		req.set(nil, errors.Annotatef(err, "retrieving transport conn: %s", streamID))
		return req
	}

	t.createTransportAsync(req, conn, false)

	return req
}

func (t *TransportFactory) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for streamID, transport := range t.transports {
		transport.Close()
		delete(t.transports, streamID)
	}

	return nil
}

type TransportRequest struct {
	cancel func()

	context      context.Context
	streamID     string
	responseChan chan TransportResponse
	torndown     chan struct{}
	setChan      chan TransportResponse
}

type TransportResponse struct {
	Transport *StreamTransport
	Err       error
}

func NewTransportRequest(ctx context.Context, streamID string) *TransportRequest {
	ctx, cancel := context.WithCancel(ctx)

	t := &TransportRequest{
		context:      ctx,
		cancel:       cancel,
		streamID:     streamID,
		responseChan: make(chan TransportResponse, 1),
		torndown:     make(chan struct{}),
		setChan:      make(chan TransportResponse),
	}

	go t.start(ctx)

	return t
}

func (t *TransportRequest) Context() context.Context {
	return t.context
}

func (t *TransportRequest) Cancel() {
	t.cancel()
}

func (t *TransportRequest) StreamID() string {
	return t.streamID
}

func (t *TransportRequest) start(ctx context.Context) {
	defer close(t.torndown)

	select {
	case <-ctx.Done():
		t.responseChan <- TransportResponse{
			Err:       errors.Trace(ctx.Err()),
			Transport: nil,
		}
	case res := <-t.setChan:
		t.responseChan <- res
	}
}

func (t *TransportRequest) set(streamTransport *StreamTransport, err error) {
	res := TransportResponse{
		Transport: streamTransport,
		Err:       err,
	}

	select {
	case t.setChan <- res:
	case <-t.torndown:
	}
}

func (t *TransportRequest) Response() <-chan TransportResponse {
	return t.responseChan
}

func (t *TransportRequest) Done() <-chan struct{} {
	return t.torndown
}
