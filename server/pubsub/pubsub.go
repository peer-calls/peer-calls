package pubsub

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

// PubSub keeps a record of all published tracks and subscriptions to them.
// The user of this implementation must implement locking if it will be used
// by multiple goroutines.
type PubSub struct {
	eventsChan chan PubTrackEvent

	events *events

	// pubs is a map of pubs indexed by clientID of transport that published the
	// track and the track SSRC.
	pubs map[clientTrack]*pub

	// pubsByPubClientID is a map of a set of pubs that have been created by a
	// particular transport (indexes by clientID).
	pubsByPubClientID map[string]pubSet

	// subsBySubClientID is a map of a set of pubs that the transport has
	// subscribed to.
	subsBySubClientID map[string]map[uint32]*pub
}

// New returns a new instance of PubSub.
func New() *PubSub {
	eventsChan := make(chan PubTrackEvent)

	return &PubSub{
		eventsChan:        eventsChan,
		events:            newEvents(eventsChan, 0),
		pubs:              map[clientTrack]*pub{},
		pubsByPubClientID: map[string]pubSet{},
		subsBySubClientID: map[string]map[uint32]*pub{},
	}
}

// Pub publishes a track.
func (p *PubSub) Pub(pubClientID string, track transport.Track) {
	clientTrack := clientTrack{
		ClientID: pubClientID,
		SSRC:     track.SSRC(),
	}

	pb := newPub(pubClientID, track)

	p.pubs[clientTrack] = pb
	if _, ok := p.pubsByPubClientID[pubClientID]; !ok {
		p.pubsByPubClientID[pubClientID] = pubSet{}
	}

	p.pubsByPubClientID[pubClientID][pb] = struct{}{}

	p.eventsChan <- PubTrackEvent{
		PubTrack: newPubTrack(pb),
		Type:     transport.TrackEventTypeAdd,
	}
}

// Unpub unpublishes a track as well as unsubs all subscribers.
func (p *PubSub) Unpub(pubClientID string, ssrc uint32) {
	track := clientTrack{
		ClientID: pubClientID,
		SSRC:     ssrc,
	}

	if pb, ok := p.pubs[track]; ok {
		for subClientID := range pb.subscribers() {
			_ = p.unsub(subClientID, pb)
		}

		delete(p.pubsByPubClientID[pubClientID], pb)

		if len(p.pubsByPubClientID[pubClientID]) == 0 {
			delete(p.pubsByPubClientID, pubClientID)
		}

		delete(p.pubs, track)

		p.eventsChan <- PubTrackEvent{
			PubTrack: newPubTrack(pb),
			Type:     transport.TrackEventTypeRemove,
		}
	}
}

// Sub subscribes to a published track.
func (p *PubSub) Sub(pubClientID string, ssrc uint32, transport Transport) error {
	track := clientTrack{
		ClientID: pubClientID,
		SSRC:     ssrc,
	}

	if pubClientID == transport.ClientID() {
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
		p.subsBySubClientID[subClientID] = map[uint32]*pub{}
	}

	p.subsBySubClientID[subClientID][pb.track.SSRC()] = pb

	return nil
}

// Unsub unsubscribes from a published track.
func (p *PubSub) Unsub(pubClientID string, ssrc uint32, subClientID string) error {
	clientTrack := clientTrack{
		ClientID: pubClientID,
		SSRC:     ssrc,
	}

	var err error

	pb, ok := p.pubs[clientTrack]
	if !ok {
		err = errors.Annotatef(ErrTrackNotFound, "unsub: track: %s, clientID: %s", clientTrack, subClientID)
	} else {
		err = errors.Annotatef(p.unsub(subClientID, pb), "unsub: track: %s, clientID: %s", clientTrack, subClientID)
	}

	return errors.Trace(err)
}

// unsub caller must hold the lock.
func (p *PubSub) unsub(subClientID string, pb *pub) error {
	err := pb.unsub(subClientID)

	delete(p.subsBySubClientID[subClientID], pb.track.SSRC())

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

	for _, pb := range p.subsBySubClientID[clientID] {
		_ = p.unsub(clientID, pb)
	}
}

// Subscribers returns all transports subscribed to a specific clientID/track
// pair.
func (p *PubSub) Subscribers(pubClientID string, ssrc uint32) []Transport {
	clientTrack := clientTrack{
		ClientID: pubClientID,
		SSRC:     ssrc,
	}

	var transports []Transport

	if pb, ok := p.pubs[clientTrack]; ok {
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

func (p *PubSub) PubClientID(subClientID string, ssrc uint32) (string, bool) {
	pb, ok := p.subsBySubClientID[subClientID][ssrc]
	if !ok {
		return "", false
	}

	return pb.clientID, true
}

// Tracks returns all published track information. The order is undefined.
func (p *PubSub) Tracks() []PubTrack {
	var ret []PubTrack

	if l := len(p.pubs); l > 0 {
		ret = make([]PubTrack, 0, l)

		for _, pub := range p.pubs {
			ret = append(ret, newPubTrack(pub))
		}
	}

	return ret
}

// SubscribeToEvents creates a new subscription to track events.
func (p *PubSub) SubscribeToEvents(clientID string) (<-chan PubTrackEvent, error) {
	ch, err := p.events.Subscribe(clientID)

	return ch, errors.Annotatef(err, "sub events: clientID: %s", clientID)
}

// UnsubscribeFromEvents removes an existing subscription from track events.
func (p *PubSub) UnsubscribeFromEvents(clientID string) error {
	err := p.events.Unsubscribe(clientID)

	return errors.Annotatef(err, "unsub events: clientID: %s", clientID)
}

func (p *PubSub) Close() {
	close(p.eventsChan)
	<-p.events.torndown
}

type pubSet map[*pub]struct{}

type userIdentifiable interface {
	UserID() string
}
