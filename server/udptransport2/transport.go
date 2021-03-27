package udptransport2

import (
	"sync"

	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/servertransport"
	"github.com/peer-calls/peer-calls/server/stringmux"
	"github.com/pion/interceptor"
)

type Transport struct {
	*servertransport.Transport
	closeWriteOnce sync.Once
	streamID       string
	closeWrite     func()
}

func NewTransport(
	log logger.Logger,
	streamID string,
	mediaConn stringmux.Conn,
	dataConn stringmux.Conn,
	metadataConn stringmux.Conn,
	ceptor interceptor.Interceptor,
) *Transport {
	closeWrite := func() {
		mediaConn.CloseWrite()
		dataConn.CloseWrite()
		metadataConn.CloseWrite()
	}

	serverTransportParams := servertransport.Params{
		Log:           log,
		MediaConn:     mediaConn,
		DataConn:      dataConn,
		MetadataConn:  metadataConn,
		Interceptor:   ceptor,
		CodecRegistry: nil,
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

func (t *Transport) StreamID() string {
	return t.streamID
}
