package pubsub

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

type Reader interface {
	Track() transport.Track
	Sub(subClientID string, trackLocal transport.TrackLocal) error
	Unsub(subClientID string) error
	Subs() []string
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
		trackRemote: trackRemote,
		subs:        map[string]transport.TrackLocal{},
		onClose:     onClose,
	}

	go t.startReadLoop()
	// go t.startFeedbackLoop()

	return t
}

func (t *TrackReader) Track() transport.Track {
	return t.trackRemote.Track()
}

func (t *TrackReader) startReadLoop() {
	defer func() {
		t.mu.Lock()

		t.closed = true

		go t.onClose()

		t.mu.Unlock()
	}()

	for {
		packet, _, err := t.trackRemote.ReadRTP()
		if err == io.ErrClosedPipe {
			return
		}

		t.mu.Lock()

		for key, trackLocal := range t.subs {
			_ = packet.MarshalSize()
			if err := trackLocal.WriteRTP(packet); err == io.ErrClosedPipe {
				delete(t.subs, key)
			}
		}

		t.mu.Unlock()
	}
}

func (t *TrackReader) Sub(subClientID string, trackLocal transport.TrackLocal) error {
	var err error

	t.mu.Lock()

	_, alreadySubscribed := t.subs[subClientID]

	switch {
	case t.closed:
		err = errors.Trace(io.ErrClosedPipe)
	case alreadySubscribed:
		err = errors.Errorf("already subscribed")
	default:
		t.subs[subClientID] = trackLocal
	}

	t.mu.Unlock()

	return errors.Trace(err)
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
