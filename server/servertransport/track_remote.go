package servertransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/atomic"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/transport/packetio"
	"github.com/pion/webrtc/v3"
)

type trackRemote struct {
	buffer      *packetio.Buffer
	rid         string // TODO simulcast
	ssrc        webrtc.SSRC
	track       transport.Track
	interceptor interceptor.Interceptor

	onSub   func() error
	onUnsub func() error

	subscribed atomic.Bool

	streamInfo           *interceptor.StreamInfo
	interceptorRTPReader interceptor.RTPReader
}

func newTrackRemote(
	track transport.Track,
	ssrc webrtc.SSRC,
	rid string,
	buffer *packetio.Buffer,
	codec transport.Codec,
	ceptor interceptor.Interceptor,
	interceptorParameters codecs.InterceptorParams,
	onSub func() error,
	onUnsub func() error,
) *trackRemote {
	t := &trackRemote{
		buffer:      buffer,
		rid:         rid,
		ssrc:        ssrc,
		track:       track,
		interceptor: ceptor,
		onSub:       onSub,
		onUnsub:     onUnsub,
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

	interceptorRTPReader := ceptor.BindRemoteStream(
		t.streamInfo, interceptor.RTPReaderFunc(t.read),
	)

	t.interceptorRTPReader = interceptorRTPReader

	return t
}

var _ transport.TrackRemote = &trackRemote{}

func (t *trackRemote) Track() transport.Track {
	return t.track
}

func (t *trackRemote) SSRC() webrtc.SSRC {
	return t.ssrc
}

func (t *trackRemote) RID() string {
	return t.rid
}

func (t *trackRemote) read(in []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
	i, err := t.buffer.Read(in)

	return i, a, errors.Trace(err)
}

func (t *trackRemote) ReadRTP() (*rtp.Packet, interceptor.Attributes, error) {
	b := make([]byte, ReceiveMTU)

	i, a, err := t.interceptorRTPReader.Read(b, interceptor.Attributes{})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "read RTP")
	}

	packet := &rtp.Packet{}

	err = packet.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTP")
	}

	return packet, a, nil
}

func (t *trackRemote) Subscribe() error {
	if !t.subscribed.CompareAndSwap(true) {
		return errors.Trace(errAlreadySubscribed)
	}

	return errors.Annotate(t.onSub(), "subscribe")
}

func (t *trackRemote) Unsubscribe() error {
	if !t.subscribed.CompareAndSwap(false) {
		return errors.Trace(errNotSubscribed)
	}

	return errors.Annotate(t.onUnsub(), "unsubscribe")
}

func (t *trackRemote) Close() {
	t.interceptor.UnbindRemoteStream(t.streamInfo)
}

var (
	errAlreadySubscribed = errors.Errorf("already subscribed")
	errNotSubscribed     = errors.Errorf("not subscribed")
)
