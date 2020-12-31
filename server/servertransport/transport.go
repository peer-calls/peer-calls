package servertransport

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/peer-calls/peer-calls/server/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
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

func NewTransport(
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

	return &Transport{
		MetadataTransport: NewMetadataTransport(log, metadataConn, clientID),
		MediaTransport:    NewMediaTransport(log, mediaConn),
		DataTransport:     NewDataTransport(log, dataConn),
		clientID:          clientID,
		closeChan:         make(chan struct{}),
	}
}

var _ transport.Transport = &Transport{}

func (t *Transport) ClientID() string {
	return t.clientID
}

func (t *Transport) CloseChannel() <-chan struct{} {
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

type MediaTransport struct {
	conn   io.ReadWriteCloser
	rtpCh  chan *rtp.Packet
	rtcpCh chan rtcp.Packet // TODO change to []rtcp.Packet
	log    logger.Logger

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

var _ transport.MediaTransport = &MediaTransport{}

func NewMediaTransport(log logger.Logger, conn io.ReadWriteCloser) *MediaTransport {
	t := MediaTransport{
		conn:   conn,
		rtpCh:  make(chan *rtp.Packet),
		rtcpCh: make(chan rtcp.Packet),
		log:    log.WithNamespaceAppended("server_media_transport"),
	}

	go t.start()

	return &t
}

func (t *MediaTransport) start() {
	defer func() {
		close(t.rtcpCh)
		close(t.rtpCh)
	}()

	buf := make([]byte, ReceiveMTU)

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

func (t *MediaTransport) handle(buf []byte) error {
	if len(buf) == 0 {
		atomic.AddInt64(&t.stats.readNoData, 1)

		return errors.Trace(ErrNoData)
	}

	switch {
	case MatchRTP(buf):
		atomic.AddInt64(&t.stats.readRTPPackets, 1)

		return t.handleRTP(buf)
	case MatchRTCP(buf):
		atomic.AddInt64(&t.stats.readRTCPPackets, 1)

		return errors.Trace(t.handleRTCP(buf))
	default:
		atomic.AddInt64(&t.stats.readUnknown, 1)

		return errors.Trace(ErrUnknownPacket)
	}
}

func (t *MediaTransport) handleRTP(buf []byte) error {
	pkt := &rtp.Packet{}

	err := pkt.Unmarshal(buf)
	if err != nil {
		return errors.Annotatef(err, "unmarshal RTP")
	}

	t.rtpCh <- pkt

	return nil
}

func (t *MediaTransport) handleRTCP(buf []byte) error {
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

func (t *MediaTransport) WriteRTCP(p []rtcp.Packet) error {
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

func (t *MediaTransport) WriteRTP(p *rtp.Packet) (int, error) {
	b, err := p.Marshal()
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTP")
	}

	// TODO skip writing rtp packet when no subscribers.

	i, err := t.conn.Write(b)

	if err == nil {
		atomic.AddInt64(&t.stats.sentRTPPackets, 1)
		atomic.AddInt64(&t.stats.sentBytes, int64(i))
	}

	return i, errors.Annotatef(err, "write RTP")
}

func (t *MediaTransport) RTPChannel() <-chan *rtp.Packet {
	return t.rtpCh
}

func (t *MediaTransport) RTCPChannel() <-chan rtcp.Packet {
	return t.rtcpCh
}

func (t *MediaTransport) Close() error {
	return t.conn.Close()
}

type DataTransport struct {
	conn         io.ReadWriteCloser
	log          logger.Logger
	messagesChan chan webrtc.DataChannelMessage
}

var _ transport.DataTransport = &DataTransport{}

func NewDataTransport(log logger.Logger, conn io.ReadWriteCloser) *DataTransport {
	transport := &DataTransport{
		log:          log.WithNamespaceAppended("server_data_transport"),
		conn:         conn,
		messagesChan: make(chan webrtc.DataChannelMessage),
	}

	go transport.start()

	return transport
}

func (t *DataTransport) start() {
	defer close(t.messagesChan)

	buf := make([]byte, ReceiveMTU)

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

func (t *DataTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return t.messagesChan
}

func (t *DataTransport) Send(message webrtc.DataChannelMessage) <-chan error {
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

func (t *DataTransport) SendText(message string) error {
	b := make([]byte, 0, len(message)+1)
	// mark as string
	b = append(b, 1)
	b = append(b, message...)

	_, err := t.conn.Write(b)

	return errors.Trace(err)
}

func (t *DataTransport) Close() error {
	return t.conn.Close()
}

type MetadataTransport struct {
	clientID      string
	conn          io.ReadWriteCloser
	log           logger.Logger
	trackEventsCh chan transport.TrackEvent
	localTracks   map[uint32]transport.TrackInfo
	remoteTracks  map[uint32]transport.TrackInfo
	mu            *sync.Mutex
}

var _ transport.MetadataTransport = &MetadataTransport{}

func NewMetadataTransport(log logger.Logger, conn io.ReadWriteCloser, clientID string) *MetadataTransport {
	log = log.WithNamespaceAppended("server_metadata_transport")

	transport := &MetadataTransport{
		clientID:      clientID,
		log:           log,
		conn:          conn,
		localTracks:   map[uint32]transport.TrackInfo{},
		remoteTracks:  map[uint32]transport.TrackInfo{},
		trackEventsCh: make(chan transport.TrackEvent),
		mu:            &sync.Mutex{},
	}

	go transport.start()

	return transport
}

func (t *MetadataTransport) start() {
	defer close(t.trackEventsCh)

	buf := make([]byte, ReceiveMTU)

	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.log.Error("Read remote data", errors.Trace(err), nil)

			return
		}

		// hack because JSON does not know how to unmarshal to Track interface
		var eventJSON struct {
			TrackInfo struct {
				Track sfu.UserTrack
				Kind  webrtc.RTPCodecType
				Mid   string
			}
			Type transport.TrackEventType
		}

		err = json.Unmarshal(buf[:i], &eventJSON)
		if err != nil {
			t.log.Error("Unmarshal remote data", err, nil)

			return
		}

		trackEvent := transport.TrackEvent{
			TrackInfo: transport.TrackInfo{
				Track: eventJSON.TrackInfo.Track,
				Kind:  eventJSON.TrackInfo.Kind,
				Mid:   eventJSON.TrackInfo.Mid,
			},
			Type:     eventJSON.Type,
			ClientID: t.clientID,
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

func (t *MetadataTransport) TrackEventsChannel() <-chan transport.TrackEvent {
	return t.trackEventsCh
}

func (t *MetadataTransport) LocalTracks() []transport.TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	localTracks := make([]transport.TrackInfo, 0, len(t.localTracks))

	for _, trackInfo := range t.localTracks {
		localTracks = append(localTracks, trackInfo)
	}

	return localTracks
}

func (t *MetadataTransport) RemoteTracks() []transport.TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	remoteTracks := make([]transport.TrackInfo, 0, len(t.remoteTracks))

	for _, trackInfo := range t.remoteTracks {
		remoteTracks = append(remoteTracks, trackInfo)
	}

	return remoteTracks
}

func (t *MetadataTransport) AddTrack(track transport.Track) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackInfo := transport.TrackInfo{
		Track: track,
		Kind:  t.getCodecType(track.PayloadType()),
		Mid:   "",
	}

	t.localTracks[track.SSRC()] = trackInfo

	trackEvent := transport.TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeAdd,
		ClientID:  t.clientID,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *MetadataTransport) sendTrackEvent(trackEvent transport.TrackEvent) error {
	b, err := json.Marshal(trackEvent)
	if err != nil {
		return errors.Annotatef(err, "sendTrackEvent: marshal")
	}

	_, err = t.conn.Write(b)

	return errors.Annotatef(err, "sendTrackEvent: write")
}

func (t *MetadataTransport) getCodecType(payloadType uint8) webrtc.RTPCodecType {
	// TODO These values are dynamic and are only valid when they are set in
	// media engine _and_ when we initiate peer connections.
	if payloadType == webrtc.DefaultPayloadTypeVP8 {
		return webrtc.RTPCodecTypeVideo
	}

	return webrtc.RTPCodecTypeAudio
}

func (t *MetadataTransport) RemoveTrack(ssrc uint32) error {
	t.mu.Lock()

	trackInfo, ok := t.localTracks[ssrc]
	delete(t.localTracks, ssrc)

	t.mu.Unlock()

	if !ok {
		return errors.Errorf("remove track: not found: %d", ssrc)
	}

	trackEvent := transport.TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeRemove,
		ClientID:  t.clientID,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *MetadataTransport) Close() error {
	return t.conn.Close()
}
