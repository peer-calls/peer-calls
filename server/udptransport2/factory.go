package udptransport2

import (
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/pion/sctp"
)

const (
	streamIndexControl uint16 = iota
	streamIndexMetadata
	streamIndexData
)

type Factory struct {
	params *FactoryParams

	stringMux *stringmux.StringMux

	transportsChannel    chan *Transport
	newTransportRequests chan transportRequest

	teardown chan struct{}
	torndown chan struct{}

	streams *factoryStreams
}

type FactoryParams struct {
	Log  logger.Logger
	Conn net.Conn
	// StringMux *stringmux.StringMux
}

func NewFactory(params FactoryParams) (*Factory, error) {
	params.Log = params.Log.WithNamespaceAppended("factory")

	params.Log.Trace("NewFactory", nil)

	readChanSize := 100

	stringMux := stringmux.New(stringmux.Params{
		Log:            params.Log,
		Conn:           params.Conn,
		MTU:            uint32(servertransport.ReceiveMTU), // TODO not sure if this is ok
		ReadChanSize:   readChanSize,
		ReadBufferSize: 0,
	})

	f := &Factory{
		params: &params,

		stringMux: stringMux,

		transportsChannel:    make(chan *Transport),
		newTransportRequests: make(chan transportRequest),

		teardown: make(chan struct{}),
		torndown: make(chan struct{}),

		streams: nil,
	}

	streams, err := f.init()
	if err != nil {
		return nil, errors.Trace(err)
	}

	f.streams = streams

	go f.start()

	return f, nil
}

func (f *Factory) init() (*factoryStreams, error) {
	f.params.Log.Trace("init start", nil)

	closers := make([]io.Closer, 0, 5)

	close := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i].Close()
		}
	}

	defer func() {
		if close != nil {
			close()
		}
	}()

	// FIXME stringmux never accepts anything therefore it could cause a deadlock
	// if it receives a connection with another StreamID.

	mediaConn, err := f.stringMux.GetConn("m")
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, mediaConn)

	sctpConn, err := f.stringMux.GetConn("s")
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, sctpConn)

	association, err := sctp.Client(sctp.Config{
		NetConn:              sctpConn,
		LoggerFactory:        pionlogger.NewFactory(f.params.Log),
		MaxMessageSize:       0,
		MaxReceiveBufferSize: 0,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, association)

	controlStream, err := association.OpenStream(streamIndexControl, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, controlStream)

	metadataStream, err := association.OpenStream(streamIndexMetadata, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, metadataStream)

	dataStream, err := association.OpenStream(streamIndexData, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, dataStream)

	laddr := f.params.Conn.LocalAddr()
	raddr := f.params.Conn.RemoteAddr()

	dataConn := newStreamConn(dataStream, laddr, raddr)
	metadataConn := newStreamConn(metadataStream, laddr, raddr)

	readBufferSize := 100

	f.params.Log.Trace("init done", nil)

	streams := &factoryStreams{
		close: nil,

		control: newControl(f.params.Log, controlStream),

		data: stringmux.New(stringmux.Params{
			Log:            f.params.Log.WithNamespaceAppended("data"),
			Conn:           dataConn,
			MTU:            uint32(servertransport.ReceiveMTU),
			ReadBufferSize: readBufferSize,
			ReadChanSize:   0,
		}),

		metadata: stringmux.New(stringmux.Params{
			Log:            f.params.Log.WithNamespaceAppended("metadata"),
			Conn:           metadataConn,
			MTU:            uint32(servertransport.ReceiveMTU),
			ReadBufferSize: readBufferSize,
			ReadChanSize:   0,
		}),

		media: stringmux.New(stringmux.Params{
			Log:            f.params.Log.WithNamespaceAppended("media"),
			Conn:           mediaConn,
			MTU:            uint32(servertransport.ReceiveMTU),
			ReadBufferSize: readBufferSize,
			ReadChanSize:   0,
		}),
	}

	closers = append(closers, streams.control)
	closers = append(closers, streams.data)
	closers = append(closers, streams.metadata)
	closers = append(closers, streams.media)

	streams.close = close

	// Do not close everything on defer.
	close = nil

	return streams, nil
}

func (f *Factory) start() {
	transports := map[string]*Transport{}

	removeTransportsChan := make(chan string)

	defer func() {
		close(f.transportsChannel)

		for streamID, transport := range transports {
			transport.Close()

			delete(transports, streamID)
		}

		f.streams.close()

		f.stringMux.Close()

		close(f.torndown)
	}()

	// TODO getOrAccept transprot

	createTransport := func(metadataConn stringmux.Conn) (*Transport, error) {
		f.params.Log.Trace("Create transport", nil)

		streamID := metadataConn.StreamID()

		dataConn, err := f.streams.data.GetConn(streamID)
		if err != nil {
			metadataConn.Close()

			return nil, errors.Trace(err)
		}

		mediaConn, err := f.streams.media.GetConn(streamID)
		if err != nil {
			metadataConn.Close()
			dataConn.Close()

			return nil, errors.Trace(err)
		}

		transport := NewTransport(f.params.Log, streamID, mediaConn, dataConn, metadataConn)

		return transport, nil
	}

	addTransport := func(streamID string, transport *Transport) {
		transports[streamID] = transport

		go func() {
			select {
			case <-transport.Done():
			case <-f.torndown:
				return
			}

			select {
			case removeTransportsChan <- streamID:
			case <-f.torndown:
			}
		}()
	}

	_handleTransportRequest := func(streamID string) (*Transport, error) {
		metadataConn, err := f.streams.metadata.GetConn(streamID)
		if err != nil {
			return nil, errors.Trace(err)
		}

		transport, err := createTransport(metadataConn)
		if err != nil {
			return nil, errors.Trace(err)
		}

		addTransport(streamID, transport)

		return transport, nil
	}

	handleTransportRequest := func(req transportRequest) {
		transport, err := _handleTransportRequest(req.streamID)

		req.res <- transportResponse{
			transport: transport,
			err:       err,
		}

		close(req.res)
	}

	acceptOrGet := func(t *Transport) bool {
		streamID := t.StreamID()

		for {
			select {
			case f.transportsChannel <- t:
				addTransport(streamID, t)

				return true
			case req := <-f.newTransportRequests:
				if req.streamID != streamID {
					handleTransportRequest(req)

					continue
				}

				req.res <- transportResponse{
					transport: t,
					err:       nil,
				}

				close(req.res)

				return true
			case <-f.teardown:
				return false
			}
		}
	}

	handleMetadataConn := func(metadataConn stringmux.Conn) bool {
		transport, err := createTransport(metadataConn)
		if err != nil {
			f.params.Log.Error("Create transport", errors.Trace(err), nil)

			return true
		}

		if !acceptOrGet(transport) {
			return false
		}

		return true
	}

	for {
		select {
		case streamID := <-removeTransportsChan:
			delete(transports, streamID)
		case req := <-f.newTransportRequests:
			handleTransportRequest(req)
		// case c, ok := <-f.streams.data.Conns():
		// 	if !ok {
		// 		return
		// 	}

		// 	c.Close()
		// case c, ok := <-f.streams.media.Conns():
		// 	if !ok {
		// 		return
		// 	}

		// 	c.Close()
		case metadataConn, ok := <-f.streams.metadata.Conns():
			if !ok {
				f.params.Log.Warn("Metadata stringmux closed", nil)

				return
			}

			// Only interested about metadata for now. The data channel only contains
			// messages which the peers won't see anyway until the transport is
			// created. Media won't be sent until someone subscribes after seeing the
			// metadata published.

			if !handleMetadataConn(metadataConn) {
				return
			}
		case <-f.teardown:
			return
		}
	}
}

func (f *Factory) TransportsChannel() <-chan *Transport {
	return f.transportsChannel
}

func (f *Factory) NewTransport(streamID string) (*Transport, error) {
	f.params.Log.Trace("NewTransport start", logger.Ctx{
		"stream_id": streamID,
	})

	defer f.params.Log.Trace("NewTransport done", logger.Ctx{
		"stream_id": streamID,
	})

	req := transportRequest{
		streamID: streamID,
		res:      make(chan transportResponse, 1),
	}

	select {
	case f.newTransportRequests <- req:
		transport, err := (<-req.res).Result()

		return transport, errors.Trace(err)
	case <-f.torndown:
		return nil, errors.Trace(io.ErrClosedPipe)
	}
}

func (f *Factory) Done() <-chan struct{} {
	return f.torndown
}

func (f *Factory) Close() {
	select {
	case f.teardown <- struct{}{}:
		<-f.torndown
	case <-f.torndown:
	}
}

type factoryStreams struct {
	close func()

	control *control

	data     *stringmux.StringMux
	metadata *stringmux.StringMux
	media    *stringmux.StringMux
}

type transportRequest struct {
	streamID string
	res      chan transportResponse
}

type transportResponse struct {
	transport *Transport
	err       error
}

func (t transportResponse) Result() (*Transport, error) {
	return t.transport, errors.Trace(t.err)
}
