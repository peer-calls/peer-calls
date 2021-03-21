package pubsub

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type Reader interface {
	Track() transport.Track
	Sub(subClientID string, trackLocal transport.TrackLocal) error
	Unsub(subClientID string) error
	Subs() []string

	SSRC() webrtc.SSRC
	RID() string
}

type TrackReader struct {
	mu      sync.Mutex
	closed  bool
	onClose func()

	trackRemote transport.TrackRemote
	subs        map[string]transport.TrackLocal
}

var _ Reader = &TrackReader{}

func NewTrackReader(trackRemote transport.TrackRemote, onClose func()) *TrackReader {
	t := &TrackReader{
		onClose: onClose,

		trackRemote: trackRemote,
		subs:        map[string]transport.TrackLocal{},
	}

	go t.startReadLoop()

	return t
}

func (t *TrackReader) Track() transport.Track {
	return t.trackRemote.Track()
}

func (t *TrackReader) startReadLoop() {
	for {
		packet, _, err := t.trackRemote.ReadRTP()
		if err != nil {
			// TODO log if not io.EOF
			break
		}

		t.mu.Lock()

		// TODO risk for deadlock on panic, mutex won't get unlocked and we'd end
		// up in defer which tries to acquire the lock again.

		for key, trackLocal := range t.subs {
			_ = packet.MarshalSize()

			if err := trackLocal.WriteRTP(packet); err == io.ErrClosedPipe {
				delete(t.subs, key)
			}

		}

		t.mu.Unlock()
	}

	t.mu.Lock()

	t.closed = true

	go t.onClose()

	t.mu.Unlock()
}

func (t *TrackReader) Sub(subClientID string, trackLocal transport.TrackLocal) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, alreadySubscribed := t.subs[subClientID]

	if t.closed {
		return errors.Trace(io.ErrClosedPipe)
	}

	if alreadySubscribed {
		return errors.Errorf("already subscribed")
	}

	t.subs[subClientID] = trackLocal

	return nil
}

func (t *TrackReader) Unsub(subClientID string) error {
	var err error

	t.mu.Lock()

	if _, ok := t.subs[subClientID]; !ok {
		err = errors.Errorf("track not found: %v", subClientID)
	} else {
		delete(t.subs, subClientID)
	}

	t.mu.Unlock()

	return errors.Trace(err)
}

func (t *TrackReader) Subs() []string {
	subs := make([]string, len(t.subs))

	t.mu.Lock()

	i := -1
	for k := range t.subs {
		i++
		subs[i] = k
	}

	t.mu.Unlock()

	return subs
}

func (t *TrackReader) SSRC() webrtc.SSRC {
	return t.trackRemote.SSRC()
}

func (t *TrackReader) RID() string {
	return t.trackRemote.RID()
}
