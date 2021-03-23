package servertransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/atomic"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/transport/packetio"
	"github.com/pion/webrtc/v3"
)

type trackRemote struct {
	buffer *packetio.Buffer
	rid    string // TODO simulcast
	ssrc   webrtc.SSRC
	track  transport.Track

	onSub   func() error
	onUnsub func() error

	subscribed atomic.Bool
}

// TODO we'll get track events but without SSRC. How to associate SSRC packets
// with TrackID??
// Maybe send t

func newTrackRemote(
	track transport.Track,
	ssrc webrtc.SSRC,
	rid string,
	buffer *packetio.Buffer,
	onSub func() error,
	onUnsub func() error,
) *trackRemote {
	return &trackRemote{
		buffer:  buffer,
		rid:     rid,
		ssrc:    ssrc,
		track:   track,
		onSub:   onSub,
		onUnsub: onUnsub,
	}
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

func (t *trackRemote) ReadRTP() (*rtp.Packet, interceptor.Attributes, error) {
	b := make([]byte, ReceiveMTU)

	i, err := t.buffer.Read(b)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "read RTP")
	}

	packet := &rtp.Packet{}

	err = packet.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTP")
	}

	// TODO interceptors

	return packet, nil, nil
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

var (
	errAlreadySubscribed = errors.Errorf("already subscribed")
	errNotSubscribed     = errors.Errorf("not subscribed")
)
