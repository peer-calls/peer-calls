package pubsub

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/transport"
)

// PubSub keeps a record of all published tracks and subscriptions to them.
// The user of this implementation must implement locking if it will be used
// by multiple goroutines.
type PubSub struct {
	log logger.Logger

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
	subsBySubClientID map[string]map[transport.TrackID]*pub
}

// New returns a new instance of PubSub.
func New(log logger.Logger) *PubSub {
	eventsChan := make(chan PubTrackEvent)

	return &PubSub{
		log:               log.WithNamespaceAppended("pubsub"),
		eventsChan:        eventsChan,
		events:            newEvents(eventsChan, 0),
		pubs:              map[clientTrack]*pub{},
		pubsByPubClientID: map[string]pubSet{},
		subsBySubClientID: map[string]map[transport.TrackID]*pub{},
	}
}

// Pub publishes a track.
func (p *PubSub) Pub(pubClientID string, track transport.Track) {
	p.log.Trace("Pub", logger.Ctx{
		"client_id": pubClientID,
		"track_id":  track.UniqueID(),
	})

	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  track.UniqueID(),
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
func (p *PubSub) Unpub(pubClientID string, trackID transport.TrackID) {
	p.log.Trace("Unpub", logger.Ctx{
		"client_id": pubClientID,
		"track_id":  trackID,
	})

	track := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
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
func (p *PubSub) Sub(pubClientID string, trackID transport.TrackID, transport Transport) error {
	p.log.Trace("Sub", logger.Ctx{
		"client_id":     transport.ClientID(),
		"track_id":      trackID,
		"pub_client_id": pubClientID,
	})

	track := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
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

func (p *PubSub) sub(pb *pub, tr Transport) error {
	subClientID := tr.ClientID()

	if err := pb.sub(subClientID, tr); err != nil {
		return errors.Trace(err)
	}

	if _, ok := p.subsBySubClientID[subClientID]; !ok {
		p.subsBySubClientID[subClientID] = map[transport.TrackID]*pub{}
	}

	p.subsBySubClientID[subClientID][pb.track.UniqueID()] = pb

	return nil
}

// Unsub unsubscribes from a published track.
func (p *PubSub) Unsub(pubClientID string, trackID transport.TrackID, subClientID string) error {
	p.log.Trace("Sub", logger.Ctx{
		"client_id":     subClientID,
		"track_id":      trackID,
		"pub_client_id": pubClientID,
	})

	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
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

	delete(p.subsBySubClientID[subClientID], pb.track.UniqueID())

	if len(p.subsBySubClientID[subClientID]) == 0 {
		delete(p.subsBySubClientID, subClientID)
	}

	return errors.Trace(err)
}

// Terminate unpublishes al tracks from from a particular client, as well as
// removes any subscriptions it has.
func (p *PubSub) Terminate(clientID string) {
	p.log.Trace("Terminate", logger.Ctx{
		"client_id": clientID,
	})

	for pb := range p.pubsByPubClientID[clientID] {
		p.Unpub(clientID, pb.track.UniqueID())
	}

	for _, pb := range p.subsBySubClientID[clientID] {
		_ = p.unsub(clientID, pb)
	}
}

// Subscribers returns all transports subscribed to a specific clientID/track
// pair.
func (p *PubSub) Subscribers(pubClientID string, trackID transport.TrackID) []Transport {
	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
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

func (p *PubSub) PubClientID(subClientID string, trackID transport.TrackID) (string, bool) {
	pb, ok := p.subsBySubClientID[subClientID][trackID]
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

	p.log.Trace(fmt.Sprintf("Tracks: %d", len(ret)), nil)

	return ret
}

// SubscribeToEvents creates a new subscription to track events.
func (p *PubSub) SubscribeToEvents(clientID string) (<-chan PubTrackEvent, error) {
	p.log.Trace("SubscribeToEvents", logger.Ctx{
		"client_id": clientID,
	})

	ch, err := p.events.Subscribe(clientID)

	return ch, errors.Annotatef(err, "sub events: clientID: %s", clientID)
}

// UnsubscribeFromEvents removes an existing subscription from track events.
func (p *PubSub) UnsubscribeFromEvents(clientID string) error {
	p.log.Trace("UnsubscribeFromEvents", logger.Ctx{
		"client_id": clientID,
	})

	err := p.events.Unsubscribe(clientID)

	return errors.Annotatef(err, "unsub events: clientID: %s", clientID)
}

// Close closes the subscription channel. The caller must ensure that no
// other methods are called after close has been called.
func (p *PubSub) Close() {
	close(p.eventsChan)
	<-p.events.torndown
}

type pubSet map[*pub]struct{}

type userIdentifiable interface {
	UserID() string
}
