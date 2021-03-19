package sfu

import (
	"sync"

	"github.com/peer-calls/peer-calls/server/transport"
)

type Subscriber struct {
	mu         sync.Mutex
	subscribed map[transport.TrackID]transport.Track
}

func (s *Subscriber) AddTrack(track transport.Track) (transport.TrackLocal, error) {
	s.mu.Lock()

	s.subscribed[track.UniqueID()] = track

	s.mu.Unlock()
}

func (s *Subscriber) RemoveTrack(track transport.TrackID) error {
	return nil
}
