package server

import (
	"fmt"
	"io"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

var receiveMTU = 8192

var ErrNoData = fmt.Errorf("cannot handle empty buffer")
var ErrUnknownPacket = fmt.Errorf("unknown packet")

type ServerTransport struct {
	*ServerMetadataTransport
	*ServerMediaTransport
	*ServerDataTransport

	clientID  string
	closeChan chan struct{}
}

func NewServerTransport(
	loggerFactory LoggerFactory,
	mediaConn io.ReadWriteCloser,
	dataConn io.ReadWriteCloser,
	metadataConn io.ReadWriteCloser,
) *ServerTransport {

	mediaLogger := loggerFactory.GetLogger("servermediatransport")
	metadataLogger := loggerFactory.GetLogger("metadatatransport")
	dataLogger := loggerFactory.GetLogger("datatransport")

	return &ServerTransport{
		ServerMetadataTransport: NewServerMetadataTransport(metadataLogger, metadataConn),
		ServerMediaTransport:    NewServerMediaTransport(mediaLogger, mediaConn),
		ServerDataTransport:     NewServerDataTransport(dataLogger, dataConn),
	}
}

var _ Transport = &ServerTransport{}

func (t *ServerTransport) ClientID() string {
	return t.clientID
}

func (t *ServerTransport) CloseChannel() <-chan struct{} {
	return t.closeChan
}

func (t *ServerTransport) Close() (err error) {
	errors := make([]error, 3)

	errors[0] = t.ServerDataTransport.Close()
	errors[1] = t.ServerMediaTransport.Close()
	errors[2] = t.ServerMetadataTransport.Close()

	close(t.closeChan)

	for _, oneError := range errors {
		if oneError != nil {
			err = oneError
			break
		}
	}

	return err
}

// ServerMediaTransport is used for server to server communication. The underlying
// transport protocol is SCTP, and the following data is transferred:
//
// 1. Ordered Metadata stream on ID 0. This stream will contain Track,
//    DataChannel and application-level metadata.
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
type ServerMediaTransport struct {
	clientID string
	conn     io.ReadWriteCloser
	rtpCh    chan *rtp.Packet
	rtcpCh   chan rtcp.Packet // TODO change to []rtcp.Packet
	logger   Logger
}

var _ MediaTransport = &ServerMediaTransport{}

func NewServerMediaTransport(logger Logger, conn io.ReadWriteCloser) *ServerMediaTransport {

	t := ServerMediaTransport{
		conn:   conn,
		rtpCh:  make(chan *rtp.Packet),
		rtcpCh: make(chan rtcp.Packet),
		logger: logger,
	}

	go t.start()

	return &t
}

func (t *ServerMediaTransport) start() {
	defer func() {
		close(t.rtcpCh)
		close(t.rtpCh)
	}()

	buf := make([]byte, receiveMTU)

	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.logger.Printf("Error reading remote data: %s", err)
			return
		}

		err = t.handle(buf[:i])
		if err != nil {
			t.logger.Printf("Error handling remote data: %s", err)
		}
	}
}

func (t *ServerMediaTransport) handle(buf []byte) error {
	if len(buf) == 0 {
		return ErrNoData
	}

	switch {
	case MatchRTP(buf):
		return t.handleRTP(buf)
	case MatchRTCP(buf):
		return t.handleRTCP(buf)
	default:
		return ErrUnknownPacket
	}
}

func (t *ServerMediaTransport) handleRTP(buf []byte) error {
	pkt := &rtp.Packet{}
	err := pkt.Unmarshal(buf)
	if err != nil {
		return fmt.Errorf("Erorr unmarshalling RTP packet: %w", err)
	}
	t.rtpCh <- pkt
	return nil
}

func (t *ServerMediaTransport) handleRTCP(buf []byte) error {
	pkts, err := rtcp.Unmarshal(buf)
	if err != nil {
		return fmt.Errorf("Error unmarshalling RTCP packet: %w", err)
	}
	// TODO we should probably keep RTCP packets together.
	for _, pkt := range pkts {
		t.rtcpCh <- pkt
	}
	return nil
}

func (t *ServerMediaTransport) WriteRTCP(p []rtcp.Packet) error {
	b, err := rtcp.Marshal(p)
	if err != nil {
		return err
	}
	_, err = t.conn.Write(b)
	return err
}

func (t *ServerMediaTransport) WriteRTP(p *rtp.Packet) (int, error) {
	b, err := p.Marshal()
	if err != nil {
		return 0, err
	}
	return t.conn.Write(b)
}

func (t *ServerMediaTransport) RTPChannel() <-chan *rtp.Packet {
	return t.rtpCh
}

func (t *ServerMediaTransport) RTCPChannel() <-chan rtcp.Packet {
	return t.rtcpCh
}

func (t *ServerMediaTransport) Close() error {
	return t.conn.Close()
}

type ServerDataTransport struct {
	conn   io.ReadWriteCloser
	logger Logger
}

var _ DataTransport = &ServerDataTransport{}

func NewServerDataTransport(logger Logger, conn io.ReadWriteCloser) *ServerDataTransport {
	return &ServerDataTransport{
		logger: logger,
		conn:   conn,
	}
}

func (t *ServerDataTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	// TODO implement this
	ch := make(chan webrtc.DataChannelMessage)
	close(ch)
	return ch
}

func (t *ServerDataTransport) Send(message []byte) error {
	// TODO implement this
	return fmt.Errorf("Not implemented")
}

func (t *ServerDataTransport) SendText(message string) error {
	// TODO implement this
	return fmt.Errorf("Not implemented")
}

func (t *ServerDataTransport) Close() error {
	return t.conn.Close()
}

type ServerMetadataTransport struct {
	conn   io.ReadWriteCloser
	logger Logger
}

var _ MetadataTransport = &ServerMetadataTransport{}

func NewServerMetadataTransport(logger Logger, conn io.ReadWriteCloser) *ServerMetadataTransport {
	return &ServerMetadataTransport{
		logger: logger,
		conn:   conn,
	}
}

func (t *ServerMetadataTransport) TrackEventsChannel() <-chan TrackEvent {
	// TODO implement this
	ch := make(chan TrackEvent)
	close(ch)
	return ch
}

func (t *ServerMetadataTransport) LocalTracks() []TrackInfo {
	return nil
}

func (t *ServerMetadataTransport) RemoteTracks() []TrackInfo {
	return nil
}

func (t *ServerMetadataTransport) AddTrack(payloadType uint8, ssrc uint32, id string, label string) error {
	return nil
}

func (t *ServerMetadataTransport) RemoveTrack(ssrc uint32) error {
	return nil
}

func (t *ServerMetadataTransport) Close() error {
	return t.conn.Close()
}
