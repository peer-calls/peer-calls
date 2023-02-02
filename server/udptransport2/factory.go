package udptransport2

import (
	"io"
	"net"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/pionlogger"
	"github.com/peer-calls/peer-calls/v4/server/servertransport"
	"github.com/peer-calls/peer-calls/v4/server/stringmux"
	"github.com/pion/interceptor"
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

	transportsChannel  chan *Transport
	localControlEvents chan localControlEvent

	teardown chan struct{}
	torndown chan struct{}

	streams *factoryStreams
}

type FactoryParams struct {
	Log logger.Logger
	// Conn is the net.Conn to use for creating transports.
	Conn net.Conn
	// Clock is used for creating a ticker. A Clock interface is used to allow
	// easier mocking.
	Clock clock.Clock
	// PingTimeout is the timeout after which a Ping event will be sent.
	PingTimeout time.Duration

	InterceptorRegistry *interceptor.Registry
}

func NewFactory(params FactoryParams) (*Factory, error) {
	params.Log = params.Log.WithNamespaceAppended("factory").WithCtx(logger.Ctx{
		"local_addr":  params.Conn.LocalAddr(),
		"remote_addr": params.Conn.RemoteAddr(),
	})

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

		transportsChannel:  make(chan *Transport),
		localControlEvents: make(chan localControlEvent),

		teardown: make(chan struct{}),
		torndown: make(chan struct{}),

		streams: nil,
	}

	streams, err := f.init()
	if err != nil {
		return nil, errors.Trace(err)
	}

	f.streams = streams

	pingTicker := f.params.Clock.NewTicker(f.params.PingTimeout)

	go f.start(pingTicker)

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

	closers = append(closers, associationCloser{association})

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

		control: newControlTransport(f.params.Log, controlStream),

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

func (f *Factory) start(pingTicker clock.Ticker) {
	type transportWithTracker struct {
		transport *Transport
		tracker   *controlStateTracker
	}

	transports := map[identifiers.RoomID]*transportWithTracker{}

	defer func() {
		pingTicker.Stop()

		close(f.transportsChannel)

		for streamID, t := range transports {
			if t.transport != nil {
				t.transport.Close()
			}

			delete(transports, streamID)
		}

		f.streams.close()

		f.stringMux.Close()

		close(f.torndown)
	}()

	createTransport := func(streamID identifiers.RoomID) (*Transport, error) {
		log := f.params.Log.WithCtx(logger.Ctx{
			"stream_id": streamID,
		})

		log.Trace("Create transport", nil)

		metadataConn, err := f.streams.metadata.GetConn(streamID.String())
		if err != nil {
			return nil, errors.Trace(err)
		}

		dataConn, err := f.streams.data.GetConn(streamID.String())
		if err != nil {
			metadataConn.Close()

			return nil, errors.Trace(err)
		}

		mediaConn, err := f.streams.media.GetConn(streamID.String())
		if err != nil {
			metadataConn.Close()
			dataConn.Close()

			return nil, errors.Trace(err)
		}

		transport := NewTransport(f.params.Log, streamID, mediaConn, dataConn, metadataConn, f.params.InterceptorRegistry)

		return transport, nil
	}

	handleRemoteEvent := func(event remoteControlEvent) bool {
		streamID := event.StreamID

		log := f.params.Log.WithCtx(logger.Ctx{
			"stream_id":            streamID,
			"remote_control_event": event.Type,
		})

		log.Trace("Handle remote event", nil)

		t, ok := transports[streamID]
		if !ok {
			t = &transportWithTracker{
				transport: nil,
				tracker:   &controlStateTracker{},
			}
			transports[event.StreamID] = t
		}

		responseEvent, stateChanged, err := t.tracker.handleRemoteEvent(event.Type)
		if err != nil {
			log.Error("Invalid state change", errors.Trace(err), nil)

			// Something weird is going on, teardown.
			return false
		}

		if stateChanged {
			switch event.Type {
			case remoteControlEventTypeCreate:
				transport, err := createTransport(streamID)
				if err != nil {
					log.Error("Create transport", errors.Trace(err), nil)
				}

				t.transport = transport

				select {
				case f.transportsChannel <- transport:
				case <-f.teardown:
					return false
				}
			case remoteControlEventTypeCreateAck:
				if t.transport == nil {
					log.Error("Got create_ack but transport was nil", nil, nil)

					return false
				}

				select {
				case f.transportsChannel <- t.transport:
				case <-f.teardown:
					return false
				}
			case remoteControlEventTypeClose:
				if t.transport == nil {
					log.Error("Got create_ack but transport was nil", nil, nil)

					return false
				}

				// Call CloseWrite first to ensure no more packets are sent, because
				// Close might propagate later, in case someone is listening to Done()
				// event.
				t.transport.CloseWrite()
				t.transport.Close()
				delete(transports, streamID)
			case remoteControlEventTypeCloseAck:
				if t.transport == nil {
					log.Error("Got create_ack but transport was nil", nil, nil)

					return false
				}

				t.transport.Close()
				delete(transports, streamID)
			case remoteControlEventTypeNone:
			}
		}

		if responseEvent != remoteControlEventTypeNone {
			err := f.streams.control.Send(controlEvent{
				RemoteControlEvent: &remoteControlEvent{
					StreamID: streamID,
					Type:     responseEvent,
				},
				Ping: false,
			})
			if err != nil {
				log.Error("Send control event response", errors.Trace(err), nil)

				return false
			}
		}

		return true
	}

	handleLocalEvent := func(event localControlEvent) bool {
		streamID := event.streamID

		log := f.params.Log.WithCtx(logger.Ctx{
			"stream_id":           streamID,
			"local_control_event": event.typ,
		})

		log.Trace("Handle local event", nil)

		t, ok := transports[streamID]
		if !ok {
			t = &transportWithTracker{
				transport: nil,
				tracker:   &controlStateTracker{},
			}
			transports[streamID] = t
		}

		remoteEvent := t.tracker.handleLocalEvent(event.typ)

		// nolint:exhaustive
		switch remoteEvent {
		case remoteControlEventTypeCreate:
			transport, err := createTransport(streamID)
			if err != nil {
				log.Error("Create transport", errors.Trace(err), nil)

				// Major error, teardown.
				return false
			}

			t.transport = transport
		case remoteControlEventTypeClose:
			if t.transport == nil {
				log.Error("Want close but transport is nil", nil, nil)

				return false
			}

			t.transport.CloseWrite()
		}

		if remoteEvent != remoteControlEventTypeNone {
			err := f.streams.control.Send(controlEvent{
				RemoteControlEvent: &remoteControlEvent{
					StreamID: streamID,
					Type:     remoteEvent,
				},
				Ping: false,
			})
			if err != nil {
				log.Error("Send remote control event", errors.Trace(err), nil)

				return false
			}
		}

		return true
	}

	handleUnexpectedConn := func(conn stringmux.Conn, typ string, ok bool) bool {
		if !ok {
			f.params.Log.Warn("stream closed", logger.Ctx{
				"conn_type": typ,
			})

			return false
		}

		f.params.Log.Warn("Unexpected conn", logger.Ctx{
			"conn_type": typ,
			"stream_id": conn.StreamID(),
		})

		return true
	}

	for {
		select {
		case <-pingTicker.C():
			err := f.streams.control.Send(controlEvent{
				RemoteControlEvent: nil,
				Ping:               true,
			})
			if err != nil {
				f.params.Log.Error("Send ping", errors.Trace(err), nil)

				return
			}
		case event, ok := <-f.streams.control.Events():
			if !ok {
				return
			}

			if rce := event.RemoteControlEvent; rce != nil && !handleRemoteEvent(*rce) {
				return
			}
		case event := <-f.localControlEvents:
			if !handleLocalEvent(event) {
				return
			}
		case c, ok := <-f.streams.data.Conns():
			if !handleUnexpectedConn(c, "data", ok) {
				return
			}
		case c, ok := <-f.streams.media.Conns():
			if !handleUnexpectedConn(c, "media", ok) {
				return
			}
		case c, ok := <-f.streams.metadata.Conns():
			if !handleUnexpectedConn(c, "metadata", ok) {
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

func (f *Factory) CreateTransport(streamID identifiers.RoomID) error {
	f.params.Log.Trace("CreateTransport", logger.Ctx{
		"stream_id": streamID,
	})

	event := localControlEvent{
		typ:      localControlEventTypeWantCreate,
		streamID: streamID,
	}

	select {
	case f.localControlEvents <- event:
		return nil
	case <-f.torndown:
		return errors.Annotatef(io.ErrClosedPipe, "create transport: %s", streamID)
	}
}

func (f *Factory) CloseTransport(streamID identifiers.RoomID) error {
	f.params.Log.Trace("CloseTransport", logger.Ctx{
		"stream_id": streamID,
	})

	event := localControlEvent{
		typ:      localControlEventTypeWantClose,
		streamID: streamID,
	}

	select {
	case f.localControlEvents <- event:
		return nil
	case <-f.torndown:
		return errors.Annotatef(io.ErrClosedPipe, "close transport: %s", streamID)
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

	control *controlTransport

	data     *stringmux.StringMux
	metadata *stringmux.StringMux
	media    *stringmux.StringMux
}

type associationCloser struct {
	association *sctp.Association
}

func (a associationCloser) Close() error {
	a.association.Abort("tearing down")
	return nil
}
