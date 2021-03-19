package pubsub

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/transport"
)

// PubSub keeps a record of all published tracks and subscriptions to them.
// The user of this implementation must implement locking if it will be used
// by multiple goroutines.
type PubSub struct {
	log logger.Logger

	eventsChan chan PubTrackEvent

	events *events

	// readers is a map of readers indexed by clientID of transport that
	// published the track and the track SSRC.
	readers map[clientTrack]Reader

	// readersByPubClientID is a map of a set of pubs that have been created by a
	// particular transport (indexes by clientID).
	readersByPubClientID map[string]readerSet

	// subsBySubClientID is a map of a set of readers that the transport has
	// subscribed to.
	subsBySubClientID map[string]subscriber
}

type subscriber struct {
	transport      Transport
	readersByTrack map[transport.TrackID]Reader
}

// New returns a new instance of PubSub.
func New(log logger.Logger) *PubSub {
	eventsChan := make(chan PubTrackEvent)

	return &PubSub{
		log:                  log.WithNamespaceAppended("pubsub"),
		eventsChan:           eventsChan,
		events:               newEvents(eventsChan, 0),
		readers:              map[clientTrack]Reader{},
		readersByPubClientID: map[string]readerSet{},
		subsBySubClientID:    map[string]subscriber{},
	}
}

// Pub publishes a track.
func (p *PubSub) Pub(pubClientID string, reader Reader) {
	track := reader.Track()

	p.log.Trace("Pub", logger.Ctx{
		"client_id": pubClientID,
		"track_id":  track.UniqueID(),
	})

	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  track.UniqueID(),
	}

	p.readers[clientTrack] = reader
	if _, ok := p.readersByPubClientID[pubClientID]; !ok {
		p.readersByPubClientID[pubClientID] = readerSet{}
	}

	p.readersByPubClientID[pubClientID][reader] = struct{}{}

	p.eventsChan <- PubTrackEvent{
		PubTrack: newPubTrack(pubClientID, track),
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

	if reader, ok := p.readers[track]; ok {
		for _, subClientID := range reader.Subs() {
			_ = p.unsub(subClientID, reader)
		}

		delete(p.readersByPubClientID[pubClientID], reader)

		if len(p.readersByPubClientID[pubClientID]) == 0 {
			delete(p.readersByPubClientID, pubClientID)
		}

		delete(p.readers, track)

		p.eventsChan <- PubTrackEvent{
			PubTrack: newPubTrack(pubClientID, reader.Track()),
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

	reader, ok := p.readers[track]
	if !ok {
		err = errors.Annotatef(ErrTrackNotFound, "sub: track: %s, clientID: %s", track, transport.ClientID())
	} else {
		err = errors.Annotatef(p.sub(reader, transport), "sub: track: %s, clientID: %s", track, transport.ClientID())
	}

	return errors.Trace(err)
}

func (p *PubSub) sub(reader Reader, tr Transport) error {
	subClientID := tr.ClientID()

	track := reader.Track()

	trackLocal, err := tr.AddTrack(track)
	if err != nil {
		return errors.Annotatef(err, "adding track to transport")
	}

	if err := reader.Sub(subClientID, trackLocal); err != nil {
		// TODO what to do with the track now?
		return errors.Trace(err)
	}

	if _, ok := p.subsBySubClientID[subClientID]; !ok {
		p.subsBySubClientID[subClientID] = subscriber{
			transport:      tr,
			readersByTrack: map[transport.TrackID]Reader{},
		}
	}

	p.subsBySubClientID[subClientID].readersByTrack[track.UniqueID()] = reader

	return nil
}

// Unsub unsubscribes from a published track.
func (p *PubSub) Unsub(pubClientID string, trackID transport.TrackID, subClientID string) error {
	p.log.Trace("Unsub", logger.Ctx{
		"client_id":     subClientID,
		"track_id":      trackID,
		"pub_client_id": pubClientID,
	})

	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
	}

	var err error

	reader, ok := p.readers[clientTrack]
	if !ok {
		return errors.Annotatef(ErrTrackNotFound, "unsub: track: %s, clientID: %s", clientTrack, subClientID)
	}

	if err := p.unsub(subClientID, reader); err != nil {
		return errors.Annotatef(err, "unsub: track: %s, clientID: %s", clientTrack, subClientID)
	}

	return errors.Trace(err)
}

// unsub caller must hold the lock.
func (p *PubSub) unsub(subClientID string, reader Reader) error {
	var multiErr multierr.MultiErr

	trackID := reader.Track().UniqueID()

	err := reader.Unsub(subClientID)
	multiErr.Add(errors.Trace(err))

	sub, ok := p.subsBySubClientID[subClientID]
	if !ok {
		return errors.Annotatef(ErrSubNotFound, "subscriber not found")
	}

	err = sub.transport.RemoveTrack(trackID)
	multiErr.Add(errors.Trace(err))

	delete(p.subsBySubClientID[subClientID].readersByTrack, trackID)

	if len(p.subsBySubClientID[subClientID].readersByTrack) == 0 {
		delete(p.subsBySubClientID, subClientID)
	}

	return errors.Trace(multiErr.Err())
}

// Terminate unpublishes al tracks from from a particular client, as well as
// removes any subscriptions it has.
func (p *PubSub) Terminate(clientID string) {
	p.log.Trace("Terminate", logger.Ctx{
		"client_id": clientID,
	})

	for reader := range p.readersByPubClientID[clientID] {
		p.Unpub(clientID, reader.Track().UniqueID())
	}

	for _, reader := range p.subsBySubClientID[clientID].readersByTrack {
		_ = p.unsub(clientID, reader)
	}
}

// Subscribers returns all subscribed subClientIDs to a specific clientID/track
// pair.
func (p *PubSub) Subscribers(pubClientID string, trackID transport.TrackID) []string {
	clientTrack := clientTrack{
		ClientID: pubClientID,
		TrackID:  trackID,
	}

	var ret []string

	if reader, ok := p.readers[clientTrack]; ok {
		subs := reader.Subs()

		if l := len(subs); l > 0 {
			ret = make([]string, l)

			for i, t := range subs {
				ret[i] = t
			}
		}
	}

	return ret
}

// FIXME pion3 this was unused. figure out if it is useful.
// func (p *PubSub) PubClientID(subClientID string, trackID transport.TrackID) (string, bool) {
// 	reader, ok := p.subsBySubClientID[subClientID][trackID]
// 	if !ok {
// 		return "", false
// 	}

// 	return reader.clientID, true
// }

// Tracks returns all published track information. The order is undefined.
func (p *PubSub) Tracks() []PubTrack {
	var ret []PubTrack

	if l := len(p.readers); l > 0 {
		ret = make([]PubTrack, 0, l)

		for key, reader := range p.readers {
			ret = append(ret, newPubTrack(key.ClientID, reader.Track()))
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

type readerSet map[Reader]struct{}

type userIdentifiable interface {
	UserID() string
}
