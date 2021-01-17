package servertransport

import (
	"io"
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type MediaTransport struct {
	conn   io.ReadWriteCloser
	rtpCh  chan *rtp.Packet
	rtcpCh chan []rtcp.Packet
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
		rtcpCh: make(chan []rtcp.Packet),
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

	t.rtcpCh <- pkts

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

func (t *MediaTransport) RTCPChannel() <-chan []rtcp.Packet {
	return t.rtcpCh
}

func (t *MediaTransport) Close() error {
	return t.conn.Close()
}
