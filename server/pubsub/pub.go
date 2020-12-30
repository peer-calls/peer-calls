package pubsub

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

type transportsMap map[string]Transport

type pub struct {
	clientID string
	track    transport.Track
	subs     transportsMap
}

func newPub(clientID string, track transport.Track) *pub {
	return &pub{
		clientID: clientID,
		track:    track,
		subs:     transportsMap{},
	}
}

func (p *pub) sub(clientID string, transport Transport) error {
	if err := transport.AddTrack(p.track); err != nil {
		return errors.Trace(err)
	}

	p.subs[clientID] = transport

	return nil
}

func (p *pub) unsub(subClientID string) error {
	transport, ok := p.subs[subClientID]
	if !ok {
		return errors.Trace(ErrSubNotFound)
	}

	delete(p.subs, subClientID)

	err := transport.RemoveTrack(p.track.SSRC())

	return errors.Trace(err)
}

func (p *pub) subscribers() transportsMap {
	return p.subs
}
