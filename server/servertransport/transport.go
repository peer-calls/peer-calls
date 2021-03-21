package servertransport

import (
	"fmt"
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/peer-calls/peer-calls/server/uuid"
)

const ReceiveMTU int = 8192

var (
	ErrNoData        = errors.Errorf("cannot handle empty buffer")
	ErrUnknownPacket = errors.Errorf("unknown packet")
)

// Transport is used for server to server communication. The underlying
// transport protocol is SCTP, and the following data is transferred:
//
// 1. Ordered Metadata stream on ID 0. This stream will contain track, as well
//    as application level metadata.
// 2. Unordered Media (RTP and RTCP) streams will use odd numbered stream IDs,
//    starting from 1.
// 3. Ordered DataChannel messages on even stream IDs, starting from 2.
//
// A single Media stream transports all RTP and RTCP packets for a single
// room, and a single DataChannel stream will transport all datachannel
// messages for a single room.
//
// A single SCTP connection can be used to transport packets from multiple
// rooms. Each room will take exactly one Media stream and one DataChannel
// stream. Following the rules above, the stream IDs for a specific room
// will always be N and N+1, but the metadata for all rooms will be sent on
// stream 0.
//
// Track metadata is JSON encoded.
//
// TODO subject to change
type Transport struct {
	*MetadataTransport
	*MediaTransport
	*DataTransport

	clientID  string
	closeChan chan struct{}
	closeOnce sync.Once
}

func New(
	log logger.Logger,
	mediaConn io.ReadWriteCloser,
	dataConn io.ReadWriteCloser,
	metadataConn io.ReadWriteCloser,
) *Transport {
	clientID := fmt.Sprintf("node:" + uuid.New())
	log = log.WithNamespaceAppended("server_transport").WithCtx(logger.Ctx{
		"client_id": clientID,
	})
	log.Info("NewTransport", nil)

	mediaTransport := NewMediaTransport(log, mediaConn)

	return &Transport{
		MetadataTransport: NewMetadataTransport(log, metadataConn, mediaTransport, clientID),
		MediaTransport:    mediaTransport,
		DataTransport:     NewDataTransport(log, dataConn),
		clientID:          clientID,
		closeChan:         make(chan struct{}),
	}
}

var _ transport.Transport = &Transport{}

func (t *Transport) ClientID() string {
	return t.clientID
}

func (t *Transport) Done() <-chan struct{} {
	return t.closeChan
}

func (t *Transport) Close() (err error) {
	errs := multierr.New()

	errs.Add(t.DataTransport.Close())
	errs.Add(t.MediaTransport.Close())
	errs.Add(t.MetadataTransport.Close())

	t.closeOnce.Do(func() {
		close(t.closeChan)
	})

	return errs.Err()
}

func (t *Transport) Type() transport.Type {
	return transport.TypeServer
}
