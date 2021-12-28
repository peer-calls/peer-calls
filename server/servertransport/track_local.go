package servertransport

import (
	"io"

	"sync/atomic"

	"github.com/juju/errors"
	atomicInternal "github.com/peer-calls/peer-calls/v4/server/atomic"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type trackLocal struct {
	track       transport.Track
	writer      io.Writer
	interceptor interceptor.Interceptor

	subscribers int64
	closed      *atomicInternal.Bool

	streamInfo           *interceptor.StreamInfo
	interceptorRTPWriter interceptor.RTPWriter
}

func newTrackLocal(
	track transport.Track,
	writer io.Writer,
	ssrc webrtc.SSRC,
	codec transport.Codec,
	ceptor interceptor.Interceptor,
	interceptorParameters codecs.InterceptorParams,
) *trackLocal {
	t := &trackLocal{
		track:       track,
		writer:      writer,
		interceptor: ceptor,
		subscribers: 0,
		closed:      &atomicInternal.Bool{},
	}

	t.streamInfo = &interceptor.StreamInfo{
		ID:                  "",
		Attributes:          nil,
		SSRC:                uint32(ssrc),
		PayloadType:         uint8(interceptorParameters.PayloadType),
		RTPHeaderExtensions: interceptorParameters.RTPHeaderExtensions,
		MimeType:            codec.MimeType,
		ClockRate:           codec.ClockRate,
		Channels:            codec.Channels,
		SDPFmtpLine:         codec.SDPFmtpLine,
		RTCPFeedback:        interceptorParameters.RTCPFeedback,
	}

	t.interceptorRTPWriter = ceptor.BindLocalStream(t.streamInfo, interceptor.RTPWriterFunc(t.write))

	return t
}

var _ transport.TrackLocal = &trackLocal{}

func (t *trackLocal) Track() transport.Track {
	return t.track
}

func (t *trackLocal) Write(b []byte) (int, error) {
	var packet *rtp.Packet

	err := packet.Unmarshal(b)
	if err != nil {
		return 0, errors.Annotatef(err, "write unmarshal RTP")
	}

	i, err := t.write(&packet.Header, packet.Payload, t.streamInfo.Attributes)

	return i, errors.Trace(err)
}

func (t *trackLocal) WriteRTP(packet *rtp.Packet) error {
	_, err := t.write(&packet.Header, packet.Payload, t.streamInfo.Attributes)

	return errors.Annotatef(err, "write RTP")
}

func (t *trackLocal) ssrc() webrtc.SSRC {
	return webrtc.SSRC(t.streamInfo.SSRC)
}

func (t *trackLocal) write(header *rtp.Header, payload []byte, a interceptor.Attributes) (int, error) {
	header.SSRC = uint32(t.streamInfo.SSRC)
	header.PayloadType = uint8(t.streamInfo.PayloadType)

	packet := &rtp.Packet{
		Header:  *header,
		Payload: payload,
	}

	b, err := packet.Marshal()
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTP")
	}

	i, err := t.writer.Write(b)

	return i, errors.Annotatef(err, "write RTP")
}

func (t *trackLocal) writePacket(packet *rtp.Packet) (int, error) {
	if t.closed.Get() {
		return 0, errors.Trace(io.ErrClosedPipe)
	}

	if !t.isSubscribed() {
		// Do not write to this track if track is closed or nobody is subscribed
		// to it.
		return 0, nil
	}

	b, err := packet.Marshal()
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTP")
	}

	t.interceptorRTPWriter.Write(&packet.Header, packet.Payload, interceptor.Attributes{})

	i, err := t.writer.Write(b)

	return i, errors.Annotatef(err, "write RTP")
}

func (t *trackLocal) isSubscribed() bool {
	return atomic.LoadInt64(&t.subscribers) > 0
}

func (t *trackLocal) subscribe() {
	atomic.AddInt64(&t.subscribers, 1)
}

func (t *trackLocal) unsubscribe() {
	atomic.AddInt64(&t.subscribers, -1)
}

func (t *trackLocal) Close() {
	t.closed.Set(true)
	t.interceptor.UnbindLocalStream(t.streamInfo)
}
