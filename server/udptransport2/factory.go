package udptransport2

import (
	"io"
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pionlogger"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/sctp"
)

type Factory struct {
	params *FactoryParams

	transportsChannel chan transport.Transport

	teardown chan struct{}
	torndown chan struct{}

	streams *factoryStreams
}

type FactoryParams struct {
	Log       logger.Logger
	StringMux *stringmux.StringMux
}

func NewFactory(params FactoryParams) (*Factory, error) {
	params.Log = params.Log.WithNamespaceAppended("udptransport_factory")

	f := &Factory{
		params: &params,

		transportsChannel: make(chan transport.Transport),

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
	closers := make([]io.Closer, 0, 5)

	close := func() {
		for i := len(closers) - 1; i >= 0; i++ {
			closers[i].Close()
		}
	}

	defer func() {
		if close != nil {
			close()
		}
	}()

	mediaStream, err := f.params.StringMux.GetConn("m")
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, mediaStream)

	sctpStream, err := f.params.StringMux.GetConn("s")
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, sctpStream)

	association, err := sctp.Client(sctp.Config{
		NetConn:              sctpStream,
		LoggerFactory:        pionlogger.NewFactory(f.params.Log),
		MaxMessageSize:       0,
		MaxReceiveBufferSize: 0,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, association)

	metadataStream, err := association.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, metadataStream)

	dataStream, err := association.OpenStream(1, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		return nil, errors.Trace(err)
	}

	closers = append(closers, dataStream)

	laddr := f.params.StringMux.LocalAddr()
	raddr := f.params.StringMux.RemoteAddr()

	streams := &factoryStreams{
		close: close,

		dataStream:     newStreamConn(dataStream, laddr, raddr),
		metadataStream: newStreamConn(metadataStream, laddr, raddr),
		mediaStream:    mediaStream,
	}

	// Do not close everything on defer.
	close = nil

	return streams, nil
}

func (f *Factory) start() {
	defer func() {
		close(f.transportsChannel)

		close(f.torndown)
	}()

	for {
		select {
		case <-f.teardown:
			return
		}
	}
}

func (f *Factory) TransportsChannel() <-chan transport.Transport {
	return f.transportsChannel
}

func (f *Factory) NewTransport(streamID string) {
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

	association *sctp.Association

	dataStream net.Conn

	metadataStream net.Conn
	// mediaStream is used to pass RTP/RTCP packets. It is multiplexed by
	// stringmux per room.
	mediaStream net.Conn
}
