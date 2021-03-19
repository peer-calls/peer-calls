package sfu

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

type Publisher struct {
	mu     sync.Mutex
	tracks map[transport.TrackID]*TrackReader
}

func NewPublisher() *Publisher {
	return &Publisher{
		tracks: map[transport.TrackID]*TrackReader{},
	}
}

func (p *Publisher) AddTrack(trackRemote transport.TrackRemote) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := trackRemote.Track().UniqueID()

	if _, ok := p.tracks[id]; ok {
		return errors.Errorf("duplicate track: %s", id)
	}

	p.tracks[id] = NewTrackReader(trackRemote, func() {
		p.RemoveTrack(id)
	})

	return nil
}

func (p *Publisher) RemoveTrack(trackID transport.TrackID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.tracks[trackID]; !ok {
		return errors.Errorf("track not found: %d", trackID)
	}

	delete(p.tracks, trackID)

	return nil
}

func (p *Publisher) Subscribe(trackID transport.TrackID, s *Subscriber) error {
	p.mu.Lock()

	reader, ok := p.tracks[trackID]

	p.mu.Unlock()

	if !ok {
		return errors.Errorf("track not found")
	}

	trackLocal, err := s.AddTrack(reader.Track())
	if err != nil {
		return errors.Trace(err)
	}

	err = reader.Sub(trackLocal)

	return errors.Trace(err)
}

func (p *Publisher) Unsubscribe(clientID string, trackID transport.TrackID) error {
	p.mu.Lock()

	reader, ok := p.tracks[trackID]

	p.mu.Unlock()

	if !ok {
		return errors.Errorf("track not found")
	}

	err := reader.Unsub(clientID, trackID)

	return errors.Trace(err)
}

// Tracks returns a new slice of transport.Tracks.
func (p *Publisher) Tracks() []transport.Track {
	p.mu.Lock()

	tracks := make([]transport.Track, len(p.tracks))

	i := -1
	for _, trackReader := range p.tracks {
		i++
		tracks[i] = trackReader.Track()
	}

	p.mu.Unlock()

	return tracks
}
