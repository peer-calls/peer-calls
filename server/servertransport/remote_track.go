package servertransport

import (
	"github.com/juju/errors"
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
}

// TODO we'll get track events but without SSRC. How to associate SSRC packets
// with TrackID??
// Maybe send t

func newTrackRemote(
	track transport.Track,
	ssrc webrtc.SSRC,
	rid string,
	buffer *packetio.Buffer,
) *trackRemote {
	return &trackRemote{
		buffer: buffer,
		rid:    rid,
		ssrc:   ssrc,
		track:  track,
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

	var packet *rtp.Packet

	err = packet.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTP")
	}

	// TODO interceptors

	return packet, nil, nil
}
