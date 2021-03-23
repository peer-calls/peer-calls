package servertransport

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/transport/packetio"
	"github.com/pion/webrtc/v3"
)

type MediaStream struct {
	conn io.ReadWriteCloser
	log  logger.Logger

	bufferFactory BufferFactory

	rtpBuffers  map[webrtc.SSRC]*packetio.Buffer
	rtcpBuffers map[webrtc.SSRC]*packetio.Buffer

	mu sync.Mutex

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

const (
	// limit RTP buffer to 1MB
	rtpBufferLimit = 1000 * 1000
	// limit RTCP buffer to 100KB
	rtcpBufferLimit = 100 * 1000
)

type BufferFactory func(packetType packetio.BufferPacketType, ssrc uint32) *packetio.Buffer

func newBuffer(packetType packetio.BufferPacketType, ssrc uint32) *packetio.Buffer {
	return packetio.NewBuffer()
}

func NewMediaStream(
	log logger.Logger,
	bufferFactory BufferFactory,
	conn io.ReadWriteCloser,
) *MediaStream {
	if bufferFactory == nil {
		bufferFactory = newBuffer
	}

	t := MediaStream{
		conn: conn,
		log:  log.WithNamespaceAppended("server_media_transport"),

		bufferFactory: bufferFactory,

		rtpBuffers:  map[webrtc.SSRC]*packetio.Buffer{},
		rtcpBuffers: map[webrtc.SSRC]*packetio.Buffer{},
	}

	go t.start()

	return &t
}

func (t *MediaStream) start() {
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

func (t *MediaStream) handle(buf []byte) error {
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

func (t *MediaStream) GetOrCreateBuffer(
	packetType packetio.BufferPacketType, ssrc webrtc.SSRC,
) *packetio.Buffer {
	switch packetType {
	case packetio.RTPBufferPacket:
		return t.getOrCreateRTPBuffer(ssrc)
	case packetio.RTCPBufferPacket:
		return t.getOrCreateRTCPBuffer(ssrc)
	default:
		panic(fmt.Sprintf("unfamiliar packet type: %d", packetType))
	}
}

func (t *MediaStream) RemoveBuffer(
	packetType packetio.BufferPacketType, ssrc webrtc.SSRC,
) {
	switch packetType {
	case packetio.RTPBufferPacket:
		t.removeRTPBuffer(ssrc)
	case packetio.RTCPBufferPacket:
		t.removeRTCPBuffer(ssrc)
	default:
		panic(fmt.Sprintf("unfamiliar packet type: %d", packetType))
	}
}

func (t *MediaStream) Writer() io.Writer {
	return t.conn
}

func (t *MediaStream) getOrCreateRTPBuffer(ssrc webrtc.SSRC) *packetio.Buffer {
	t.mu.Lock()

	buffer, ok := t.rtpBuffers[ssrc]
	if !ok {
		buffer = packetio.NewBuffer()
		buffer.SetLimitSize(rtpBufferLimit)

		t.rtpBuffers[ssrc] = buffer
	}

	t.mu.Unlock()

	return buffer
}

func (t *MediaStream) getOrCreateRTCPBuffer(ssrc webrtc.SSRC) *packetio.Buffer {
	t.mu.Lock()

	buffer, ok := t.rtcpBuffers[ssrc]
	if !ok {
		buffer = packetio.NewBuffer()
		buffer.SetLimitSize(rtcpBufferLimit)

		t.rtcpBuffers[ssrc] = buffer
	}

	t.mu.Unlock()

	return buffer
}

func (t *MediaStream) removeRTCPBuffer(ssrc webrtc.SSRC) {
	t.mu.Lock()

	b, ok := t.rtcpBuffers[ssrc]
	if ok {
		b.Close()
		delete(t.rtpBuffers, ssrc)
	}

	t.mu.Unlock()
}

func (t *MediaStream) removeRTPBuffer(ssrc webrtc.SSRC) {
	t.mu.Lock()

	b, ok := t.rtpBuffers[ssrc]
	if ok {
		b.Close()
		delete(t.rtcpBuffers, ssrc)
	}

	t.mu.Unlock()
}

func (t *MediaStream) handleRTP(buf []byte) error {
	pkt := &rtp.Packet{}

	err := pkt.Unmarshal(buf)
	if err != nil {
		return errors.Annotatef(err, "unmarshal RTP")
	}

	buffer := t.getOrCreateRTPBuffer(webrtc.SSRC(pkt.SSRC))

	_, err = buffer.Write(buf)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (t *MediaStream) handleRTCP(buf []byte) error {
	packets, err := rtcp.Unmarshal(buf)
	if err != nil {
		return errors.Trace(err)
	}

	var merr multierr.MultiErr

	for _, ssrc := range destinationSSRC(packets) {
		buffer := t.getOrCreateRTCPBuffer(webrtc.SSRC(ssrc))

		if _, err := buffer.Write(buf); err != nil {
			merr.Add(errors.Annotatef(err, "read RTCP to buffer"))
		}
	}

	return errors.Trace(merr.Err())
}

func (t *MediaStream) WriteRTCP(p []rtcp.Packet) error {
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

func (t *MediaStream) writeRTP(p *rtp.Packet) (int, error) {
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

func (t *MediaStream) Close() error {
	return t.conn.Close()
}

// create a list of Destination SSRCs
// that's a superset of all Destinations in the slice.
func destinationSSRC(pkts []rtcp.Packet) []uint32 {
	ssrcSet := make(map[uint32]struct{})
	for _, p := range pkts {
		for _, ssrc := range p.DestinationSSRC() {
			ssrcSet[ssrc] = struct{}{}
		}
	}

	out := make([]uint32, 0, len(ssrcSet))
	for ssrc := range ssrcSet {
		out = append(out, ssrc)
	}

	return out
}
