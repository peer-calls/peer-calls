package server

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const receiveMTU int = 8192

var (
	ErrNoData        = errors.Errorf("cannot handle empty buffer")
	ErrUnknownPacket = errors.Errorf("unknown packet")
)

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
	closeOnce sync.Once
}

func NewServerTransport(
	log logger.Logger,
	mediaConn io.ReadWriteCloser,
	dataConn io.ReadWriteCloser,
	metadataConn io.ReadWriteCloser,
) *ServerTransport {
	clientID := fmt.Sprintf("node:" + NewUUIDBase62())
	log = log.WithNamespaceAppended("server_transport").WithCtx(logger.Ctx{
		"client_id": clientID,
	})
	log.Info("NewServerTransport", nil)

	return &ServerTransport{
		ServerMetadataTransport: NewServerMetadataTransport(log, metadataConn),
		ServerMediaTransport:    NewServerMediaTransport(log, mediaConn),
		ServerDataTransport:     NewServerDataTransport(log, dataConn),
		clientID:                clientID,
		closeChan:               make(chan struct{}),
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
	var errs MultiErrorHandler

	errs.Add(t.ServerDataTransport.Close())
	errs.Add(t.ServerMediaTransport.Close())
	errs.Add(t.ServerMetadataTransport.Close())

	t.closeOnce.Do(func() {
		close(t.closeChan)
	})

	return errs.Err()
}

type ServerMediaTransport struct {
	clientID string
	conn     io.ReadWriteCloser
	rtpCh    chan *rtp.Packet
	rtcpCh   chan rtcp.Packet // TODO change to []rtcp.Packet
	log      logger.Logger

	stats struct {
		readBytes       int64
		readNoData      int64
		readRTPPackets  int64
		readRTCPPackets int64
		readUnknown     int64

		sentBytes       int64
		sentRTPPackets  int64
		sentRTCPPackets int64
	}
}

var _ MediaTransport = &ServerMediaTransport{}

func NewServerMediaTransport(log logger.Logger, conn io.ReadWriteCloser) *ServerMediaTransport {

	t := ServerMediaTransport{
		conn:   conn,
		rtpCh:  make(chan *rtp.Packet),
		rtcpCh: make(chan rtcp.Packet),
		log:    log.WithNamespaceAppended("server_media_transport"),
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
			t.log.Error("Read remote data", errors.Trace(err), nil)
			return
		}

		atomic.AddInt64(&t.stats.readBytes, int64(i))

		// Bytes need to be copied from the buffer because unmarshaling RTP and
		// RTCP packets will not create copies, so the raw body of these packets
		// such as RTP.Payload would be replaced before being marshaled and sent
		// downstream.
		b := make([]byte, i)

		copy(b, buf[:i])

		err = t.handle(b)

		if err != nil {
			t.log.Error("Handle remote data", errors.Trace(err), nil)
		}
	}
}

func (t *ServerMediaTransport) handle(buf []byte) error {
	if len(buf) == 0 {
		atomic.AddInt64(&t.stats.readNoData, 1)
		return ErrNoData
	}

	switch {
	case MatchRTP(buf):
		atomic.AddInt64(&t.stats.readRTPPackets, 1)
		return t.handleRTP(buf)
	case MatchRTCP(buf):
		atomic.AddInt64(&t.stats.readRTCPPackets, 1)
		return t.handleRTCP(buf)
	default:
		atomic.AddInt64(&t.stats.readUnknown, 1)
		return ErrUnknownPacket
	}
}

func (t *ServerMediaTransport) handleRTP(buf []byte) error {
	pkt := &rtp.Packet{}

	err := pkt.Unmarshal(buf)
	if err != nil {
		return errors.Annotatef(err, "unmarshal RTP")
	}

	t.rtpCh <- pkt
	return nil
}

func (t *ServerMediaTransport) handleRTCP(buf []byte) error {
	pkts, err := rtcp.Unmarshal(buf)
	if err != nil {
		return errors.Annotatef(err, "unmarshal RTCP")
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
		return errors.Annotatef(err, "marshal RTCP")
	}

	i, err := t.conn.Write(b)

	if err == nil {
		atomic.AddInt64(&t.stats.sentRTCPPackets, 1)
		atomic.AddInt64(&t.stats.sentBytes, int64(i))
	}

	return errors.Annotatef(err, "write RTCP")
}

func (t *ServerMediaTransport) WriteRTP(p *rtp.Packet) (int, error) {
	b, err := p.Marshal()
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTP")
	}

	i, err := t.conn.Write(b)

	if err == nil {
		atomic.AddInt64(&t.stats.sentRTPPackets, 1)
		atomic.AddInt64(&t.stats.sentBytes, int64(i))
	}

	return i, errors.Annotatef(err, "write RTP")
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
	log          logger.Logger
	messagesChan chan webrtc.DataChannelMessage
}

var _ DataTransport = &ServerDataTransport{}

func NewServerDataTransport(log logger.Logger, conn io.ReadWriteCloser) *ServerDataTransport {
	transport := &ServerDataTransport{
		log:          log.WithNamespaceAppended("server_data_transport"),
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
			t.log.Error("Read remote data", errors.Trace(err), nil)
			return
		}

		if i < 1 {
			t.log.Error(fmt.Sprintf("Message too short: %d", i), nil, nil)
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

func (t *ServerDataTransport) Send(message webrtc.DataChannelMessage) <-chan error {
	b := make([]byte, 0, len(message.Data)+1)

	if message.IsString {
		// Mark as string
		b = append(b, 1)
	} else {
		// Mark as binary
		b = append(b, 0)
	}

	b = append(b, message.Data...)

	_, err := t.conn.Write(b)

	errCh := make(chan error, 1)
	errCh <- err

	return errCh
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
	log           logger.Logger
	trackEventsCh chan TrackEvent
	localTracks   map[uint32]TrackInfo
	remoteTracks  map[uint32]TrackInfo
	mu            *sync.Mutex
}

var _ MetadataTransport = &ServerMetadataTransport{}

func NewServerMetadataTransport(log logger.Logger, conn io.ReadWriteCloser) *ServerMetadataTransport {
	log = log.WithNamespaceAppended("server_metadata_transport")

	transport := &ServerMetadataTransport{
		log:           log,
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
			t.log.Error("Read remote data", errors.Trace(err), nil)
			return
		}

		// hack because JSON does not know how to unmarshal to Track interface
		var eventJSON struct {
			TrackInfo struct {
				Track UserTrack
				Kind  webrtc.RTPCodecType
				Mid   string
			}
			Type TrackEventType
		}

		err = json.Unmarshal(buf[:i], &eventJSON)
		if err != nil {
			t.log.Error("Unmarshal remote data", err, nil)
			return
		}

		trackEvent := TrackEvent{
			TrackInfo: TrackInfo{eventJSON.TrackInfo.Track, eventJSON.TrackInfo.Kind, eventJSON.TrackInfo.Mid},
			Type:      eventJSON.Type,
		}

		switch trackEvent.Type {
		case transport.TrackEventTypeAdd:
			t.mu.Lock()
			t.remoteTracks[trackEvent.TrackInfo.Track.SSRC()] = trackEvent.TrackInfo
			t.mu.Unlock()
		case transport.TrackEventTypeRemove:
			t.mu.Lock()
			delete(t.remoteTracks, trackEvent.TrackInfo.Track.SSRC())
			t.mu.Unlock()
		}

		t.log.Info(fmt.Sprintf("Got track event: %+v", trackEvent), nil)

		t.trackEventsCh <- trackEvent
	}
}

func (t *ServerMetadataTransport) TrackEventsChannel() <-chan TrackEvent {
	return t.trackEventsCh
}

func (t *ServerMetadataTransport) LocalTracks() []TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	localTracks := make([]TrackInfo, 0, len(t.localTracks))

	for _, trackInfo := range t.localTracks {
		localTracks = append(localTracks, trackInfo)
	}

	return localTracks
}

func (t *ServerMetadataTransport) RemoteTracks() []TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	remoteTracks := make([]TrackInfo, 0, len(t.remoteTracks))

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
		Type:      transport.TrackEventTypeAdd,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *ServerMetadataTransport) sendTrackEvent(trackEvent TrackEvent) error {
	b, err := json.Marshal(trackEvent)

	if err != nil {
		return errors.Annotatef(err, "sendTrackEvent: marshal")
	}

	_, err = t.conn.Write(b)

	return errors.Annotatef(err, "sendTrackEvent: write")
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
		return errors.Errorf("remove track: not found: %d", ssrc)
	}

	trackEvent := TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeRemove,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *ServerMetadataTransport) Close() error {
	return t.conn.Close()
}
