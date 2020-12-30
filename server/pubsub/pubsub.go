package pubsub

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

// PubSub keeps a record of all published tracks and subscriptions to them.
// The user of this implementation must implement locking if it will be used
// by multiple goroutines.
type PubSub struct {
	// pubs is a map of pubs indexed by clientID of transport that published the
	// track and the track SSRC.
	pubs map[publishedTrack]*pub

	// pubsByPubClientID is a map of a set of pubs that have been created by a
	// particular transport (indexes by clientID).
	pubsByPubClientID map[string]pubSet

	// subsBySubClientID is a map of a set of pubs that the transport has
	// subscribed to.
	subsBySubClientID map[string]pubSet
}

// New returns a new instance of PubSub.
func New() *PubSub {
	return &PubSub{
		pubs:              map[publishedTrack]*pub{},
		pubsByPubClientID: map[string]pubSet{},
		subsBySubClientID: map[string]pubSet{},
	}
}

// Pub publishes a track.
func (p *PubSub) Pub(clientID string, track transport.Track) {
	pTrack := publishedTrack{
		clientID: clientID,
		ssrc:     track.SSRC(),
	}

	pb := newPub(clientID, track)

	p.pubs[pTrack] = pb
	if _, ok := p.pubsByPubClientID[clientID]; !ok {
		p.pubsByPubClientID[clientID] = pubSet{}
	}

	p.pubsByPubClientID[clientID][pb] = struct{}{}
}

// Unpub unpublishes a track as well as unsubs all subscribers.
func (p *PubSub) Unpub(clientID string, ssrc uint32) {
	track := publishedTrack{
		clientID: clientID,
		ssrc:     ssrc,
	}

	if pb, ok := p.pubs[track]; ok {
		for subClientID := range pb.subscribers() {
			_ = p.unsub(subClientID, pb)
		}

		delete(p.pubsByPubClientID[clientID], pb)

		if len(p.pubsByPubClientID[clientID]) == 0 {
			delete(p.pubsByPubClientID, clientID)
		}

		delete(p.pubs, track)
	}
}

// Sub subscribes to a published track.
func (p *PubSub) Sub(clientID string, ssrc uint32, transport Transport) error {
	track := publishedTrack{
		clientID: clientID,
		ssrc:     ssrc,
	}

	if clientID == transport.ClientID() {
		return errors.Annotatef(ErrSubscribeToOwnTrack, "sub: track: %s, clientID: %s", track, transport.ClientID())
	}

	var err error

	pb, ok := p.pubs[track]
	if !ok {
		err = errors.Annotatef(ErrTrackNotFound, "sub: track: %s, clientID: %s", track, transport.ClientID())
	} else {
		err = errors.Annotatef(p.sub(pb, transport), "sub: track: %s, clientID: %s", track, transport.ClientID())
	}

	return errors.Trace(err)
}

func (p *PubSub) sub(pb *pub, transport Transport) error {
	subClientID := transport.ClientID()

	if err := pb.sub(subClientID, transport); err != nil {
		return errors.Trace(err)
	}

	if _, ok := p.subsBySubClientID[subClientID]; !ok {
		p.subsBySubClientID[subClientID] = pubSet{}
	}

	p.subsBySubClientID[subClientID][pb] = struct{}{}

	return nil
}

// Unsub unsubscribes from a published track.
func (p *PubSub) Unsub(clientID string, ssrc uint32, subClientID string) error {
	track := publishedTrack{
		clientID: clientID,
		ssrc:     ssrc,
	}

	var err error

	pb, ok := p.pubs[track]
	if !ok {
		err = errors.Annotatef(ErrTrackNotFound, "unsub: track: %s, clientID: %s", track, subClientID)
	} else {
		err = errors.Annotatef(p.unsub(subClientID, pb), "unsub: track: %s, clientID: %s", track, subClientID)
	}

	return errors.Trace(err)
}

// unsub caller must hold the lock.
func (p *PubSub) unsub(subClientID string, pb *pub) error {
	err := pb.unsub(subClientID)

	delete(p.subsBySubClientID[subClientID], pb)

	if len(p.subsBySubClientID[subClientID]) == 0 {
		delete(p.subsBySubClientID, subClientID)
	}

	return errors.Trace(err)
}

// Terminate unpublishes al tracks from from a particular client, as well as
// removes any subscriptions it has.
func (p *PubSub) Terminate(clientID string) {
	for pb := range p.pubsByPubClientID[clientID] {
		p.Unpub(clientID, pb.track.SSRC())
	}

	for pb := range p.subsBySubClientID[clientID] {
		_ = p.unsub(clientID, pb)
	}
}

// Subscribers returns all transports subscribed to a specific clientID/track
// pair.
func (p *PubSub) Subscribers(clientID string, ssrc uint32) []Transport {
	pTrack := publishedTrack{
		clientID: clientID,
		ssrc:     ssrc,
	}

	var transports []Transport

	if pb, ok := p.pubs[pTrack]; ok {
		subs := pb.subscribers()

		if l := len(subs); l > 0 {
			transports = make([]Transport, 0, l)

			for _, t := range subs {
				transports = append(transports, t)
			}
		}
	}

	return transports
}

type publishedTrack struct {
	clientID string
	ssrc     uint32
}

func (p publishedTrack) String() string {
	return fmt.Sprintf("%s:%d", p.clientID, p.ssrc)
}

type pubSet map[*pub]struct{}
