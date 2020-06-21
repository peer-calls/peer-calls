package server

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

var receiveMTU int = 8192

var ErrNoData = fmt.Errorf("cannot handle empty buffer")
var ErrUnknownPacket = fmt.Errorf("unknown packet")

// ServerTransport is used for server to server communication. The underlying
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

	clientID := fmt.Sprintf("node:" + NewUUIDBase62())
	logger := loggerFactory.GetLogger("servertransport")
	logger.Printf("NewServerTransport: %s", clientID)

	mediaLogger := loggerFactory.GetLogger("servermediatransport")
	metadataLogger := loggerFactory.GetLogger("metadatatransport")
	dataLogger := loggerFactory.GetLogger("datatransport")

	return &ServerTransport{
		ServerMetadataTransport: NewServerMetadataTransport(metadataLogger, metadataConn),
		ServerMediaTransport:    NewServerMediaTransport(mediaLogger, mediaConn),
		ServerDataTransport:     NewServerDataTransport(dataLogger, dataConn),
		clientID:                clientID,
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
	conn         io.ReadWriteCloser
	logger       Logger
	messagesChan chan webrtc.DataChannelMessage
}

var _ DataTransport = &ServerDataTransport{}

func NewServerDataTransport(logger Logger, conn io.ReadWriteCloser) *ServerDataTransport {
	transport := &ServerDataTransport{
		logger:       logger,
		conn:         conn,
		messagesChan: make(chan webrtc.DataChannelMessage),
	}

	go transport.start()

	return transport
}

func (t *ServerDataTransport) start() {
	defer close(t.messagesChan)

	buf := make([]byte, receiveMTU)
	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.logger.Printf("Error reading remote data: %s", err)
			return
		}

		if i < 1 {
			t.logger.Printf("Message too short: %d", i)
			return
		}

		// This is a little wasteful as a whole byte is being used as a boolean,
		// but works for now.
		isString := !(buf[0] == 0)

		// TODO figure out which user a message belongs to.
		message := webrtc.DataChannelMessage{
			IsString: isString,
			Data:     buf[1:],
		}

		t.messagesChan <- message
	}
}

func (t *ServerDataTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return t.messagesChan
}

func (t *ServerDataTransport) Send(message []byte) error {
	b := make([]byte, 0, len(message)+1)
	// mark as binary
	b = append(b, 0)
	b = append(b, message...)

	_, err := t.conn.Write(b)
	return err
}

func (t *ServerDataTransport) SendText(message string) error {
	b := make([]byte, 0, len(message)+1)
	// mark as string
	b = append(b, 1)
	b = append(b, message...)

	_, err := t.conn.Write(b)
	return err
}

func (t *ServerDataTransport) Close() error {
	return t.conn.Close()
}

type ServerMetadataTransport struct {
	conn          io.ReadWriteCloser
	logger        Logger
	trackEventsCh chan TrackEvent
	localTracks   map[uint32]TrackInfo
	remoteTracks  map[uint32]TrackInfo
	mu            *sync.Mutex
}

var _ MetadataTransport = &ServerMetadataTransport{}

func NewServerMetadataTransport(logger Logger, conn io.ReadWriteCloser) *ServerMetadataTransport {
	transport := &ServerMetadataTransport{
		logger:        logger,
		conn:          conn,
		localTracks:   map[uint32]TrackInfo{},
		remoteTracks:  map[uint32]TrackInfo{},
		trackEventsCh: make(chan TrackEvent),
		mu:            &sync.Mutex{},
	}

	go transport.start()

	return transport
}

func (t *ServerMetadataTransport) start() {
	defer close(t.trackEventsCh)

	buf := make([]byte, receiveMTU)
	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.logger.Printf("Error reading remote data: %s", err)
			return
		}

		var trackEvent TrackEvent

		err = json.Unmarshal(buf[:i], &trackEvent)
		if err != nil {
			t.logger.Printf("Error unmarshalling remote data: %s", err)
			return
		}

		t.trackEventsCh <- trackEvent
	}
}

func (t *ServerMetadataTransport) TrackEventsChannel() <-chan TrackEvent {
	return t.trackEventsCh
}

func (t *ServerMetadataTransport) LocalTracks() []TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	localTracks := make([]TrackInfo, len(t.localTracks))

	for _, trackInfo := range t.localTracks {
		localTracks = append(localTracks, trackInfo)
	}

	return localTracks
}

func (t *ServerMetadataTransport) RemoteTracks() []TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	remoteTracks := make([]TrackInfo, len(t.remoteTracks))

	for _, trackInfo := range t.remoteTracks {
		remoteTracks = append(remoteTracks, trackInfo)
	}

	return remoteTracks
}

func (t *ServerMetadataTransport) AddTrack(track Track) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackInfo := TrackInfo{
		Track: track,
		Kind:  t.getCodecType(track.PayloadType()),
		Mid:   "",
	}

	t.localTracks[track.SSRC()] = trackInfo

	trackEvent := TrackEvent{
		TrackInfo: trackInfo,
		Type:      TrackEventTypeAdd,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *ServerMetadataTransport) sendTrackEvent(trackEvent TrackEvent) error {
	b, err := json.Marshal(trackEvent)

	if err != nil {
		return fmt.Errorf("sendTrackEvent: error marshaling trackEvent to JSON: %w", err)
	}

	_, err = t.conn.Write(b)

	return err
}

func (t *ServerMetadataTransport) getCodecType(payloadType uint8) webrtc.RTPCodecType {
	// TODO These values are dynamic and are only valid when they are set in
	// media engine _and_ when we initiate peer connections.
	if payloadType == webrtc.DefaultPayloadTypeVP8 {
		return webrtc.RTPCodecTypeVideo
	}
	return webrtc.RTPCodecTypeAudio
}

func (t *ServerMetadataTransport) RemoveTrack(ssrc uint32) error {
	t.mu.Lock()

	trackInfo, ok := t.localTracks[ssrc]
	delete(t.localTracks, ssrc)

	t.mu.Unlock()

	if !ok {
		return fmt.Errorf("RemoveTrack: Track not found: %d", ssrc)
	}

	trackEvent := TrackEvent{
		TrackInfo: trackInfo,
		Type:      TrackEventTypeRemove,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *ServerMetadataTransport) Close() error {
	return t.conn.Close()
}
