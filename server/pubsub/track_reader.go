package pubsub

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
)

type Reader interface {
	Track() transport.Track
	Sub(subClientID identifiers.ClientID, trackLocal transport.TrackLocal) error
	Unsub(subClientID identifiers.ClientID) error
	Subs() []identifiers.ClientID

	SSRC() webrtc.SSRC
	RID() string
}

type TrackReader struct {
	mu      sync.Mutex
	closed  bool
	onClose func()

	trackRemote transport.TrackRemote
	subs        map[identifiers.ClientID]transport.TrackLocal
}

var _ Reader = &TrackReader{}

func NewTrackReader(trackRemote transport.TrackRemote, onClose func()) *TrackReader {
	t := &TrackReader{
		onClose: onClose,

		trackRemote: trackRemote,
		subs:        map[identifiers.ClientID]transport.TrackLocal{},
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

		numSent := float64(0)

		for key, trackLocal := range t.subs {
			if err := trackLocal.WriteRTP(packet); err != nil {
				if multierr.Is(err, io.ErrClosedPipe) {
					_ = t.unsub(key)
				}

				continue
			}

			numSent++
		}

		t.mu.Unlock()

		packetSize := float64(packet.MarshalSize())

		prometheusRTPPacketsReceived.Inc()
		prometheusRTPPacketsReceivedBytes.Add(packetSize)
		prometheusRTPPacketsSent.Add(numSent)
		prometheusRTPPacketsSentBytes.Add(packetSize * numSent)
	}

	t.mu.Lock()

	t.closed = true

	go t.onClose()

	t.mu.Unlock()
}

func (t *TrackReader) Sub(subClientID identifiers.ClientID, trackLocal transport.TrackLocal) error {
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

	// TODO do not block network IO.
	if sub, ok := t.trackRemote.(subscribable); ok {
		_ = sub.Subscribe()
	}

	return nil
}

func (t *TrackReader) unsub(subClientID identifiers.ClientID) error {
	var err error

	if _, ok := t.subs[subClientID]; !ok {
		return errors.Errorf("track not found: %v", subClientID)
	}

	delete(t.subs, subClientID)

	// TODO do not block network IO.
	if unsub, ok := t.trackRemote.(unsubscribable); ok {
		err = unsub.Unsubscribe()
		err = errors.Annotate(err, "Unsubscribe")
	}

	return errors.Trace(err)
}

func (t *TrackReader) Unsub(subClientID identifiers.ClientID) error {
	t.mu.Lock()

	err := t.unsub(subClientID)

	t.mu.Unlock()

	return errors.Trace(err)
}

func (t *TrackReader) Subs() []identifiers.ClientID {
	subs := make([]identifiers.ClientID, len(t.subs))

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

type subscribable interface {
	Subscribe() error
}

type unsubscribable interface {
	Unsubscribe() error
}
