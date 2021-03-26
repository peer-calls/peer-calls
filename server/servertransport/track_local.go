package servertransport

import (
	"io"

	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type trackLocal struct {
	track  transport.Track
	writer io.Writer
	ssrc   webrtc.SSRC

	subscribers int64
}

func newTrackLocal(
	track transport.Track,
	writer io.Writer,
	ssrc webrtc.SSRC,
) *trackLocal {
	return &trackLocal{
		track:       track,
		writer:      writer,
		ssrc:        ssrc,
		subscribers: 0,
	}
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

	i, err := t.write(packet)

	return i, errors.Trace(err)
}

func (t *trackLocal) WriteRTP(packet *rtp.Packet) error {
	packet.Header.SSRC = uint32(t.ssrc)

	// TODO I might be wrong but I don't think we need to worry about
	// payload types here, because that will be sent in the metadata,
	// and will eventually be overwritten by pion/webrtc.

	_, err := t.write(packet)

	return errors.Annotatef(err, "write RTP")
}

func (t *trackLocal) write(packet *rtp.Packet) (int, error) {
	if !t.isSubscribed() {
		// Do not write to this track if nobody is subscribed to it.
		return 0, nil
	}

	b, err := packet.Marshal()
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTP")
	}

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
