package udptransport2

import (
	"sync"

	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/servertransport"
	"github.com/peer-calls/peer-calls/v4/server/stringmux"
	"github.com/pion/interceptor"
)

type Transport struct {
	*servertransport.Transport
	closeWriteOnce sync.Once
	streamID       identifiers.RoomID
	closeWrite     func()
}

func NewTransport(
	log logger.Logger,
	streamID identifiers.RoomID,
	mediaConn stringmux.Conn,
	dataConn stringmux.Conn,
	metadataConn stringmux.Conn,
	interceptorRegistry *interceptor.Registry,
) *Transport {
	closeWrite := func() {
		mediaConn.CloseWrite()
		dataConn.CloseWrite()
		metadataConn.CloseWrite()
	}

	serverTransportParams := servertransport.Params{
		Log:                 log,
		MediaConn:           mediaConn,
		DataConn:            dataConn,
		MetadataConn:        metadataConn,
		InterceptorRegistry: interceptorRegistry,
		CodecRegistry:       nil,
	}

	return &Transport{
		streamID:       streamID,
		Transport:      servertransport.New(serverTransportParams),
		closeWriteOnce: sync.Once{},
		closeWrite:     closeWrite,
	}
}

func (t *Transport) CloseWrite() {
	t.closeWriteOnce.Do(t.closeWrite)
}

// TODO rename to RoomID.
func (t *Transport) StreamID() identifiers.RoomID {
	return t.streamID
}
