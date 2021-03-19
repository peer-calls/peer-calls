package sfu

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

type TrackReader struct {
	mu      sync.Mutex
	closed  bool
	onClose func()

	trackRemote transport.TrackRemote
	subs        map[subKey]transport.TrackLocal
}

type subKey struct {
	UniqueID transport.TrackID
	UserID   string
}

func NewTrackReader(trackRemote transport.TrackRemote, onClose func()) *TrackReader {
	t := &TrackReader{
		trackRemote: trackRemote,
		subs:        map[subKey]transport.TrackLocal{},
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
			if _, err := trackLocal.WriteRTP(packet); err == io.ErrClosedPipe {
				delete(t.subs, key)
			}
		}

		t.mu.Unlock()
	}
}

func (t *TrackReader) Sub(trackLocal transport.TrackLocal) error {
	key := subKey{
		UniqueID: trackLocal.Track().UniqueID(),
		UserID:   trackLocal.Track().UserID(),
	}

	var err error

	t.mu.Lock()

	_, alreadySubscribed := t.subs[key]

	switch {
	case t.closed:
		err = errors.Trace(io.ErrClosedPipe)
	case alreadySubscribed:
		err = errors.Errorf("already subscribed")
	default:
		t.subs[key] = trackLocal
	}

	t.mu.Unlock()

	return errors.Trace(err)
}

func (t *TrackReader) Unsub(clientID string, trackID transport.TrackID) error {
	key := subKey{
		UniqueID: trackID,
		UserID:   clientID,
	}

	var err error

	t.mu.Lock()

	if _, ok := t.subs[key]; !ok {
		err = errors.Errorf("track not found: %v", key)
	} else {
		delete(t.subs, key)
	}

	t.mu.Unlock()

	return errors.Trace(err)
}
