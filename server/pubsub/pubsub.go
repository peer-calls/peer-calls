package pubsub

import (
	"fmt"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
)

// PubSub keeps a record of all published tracks and subscriptions to them.
// The user of this implementation must implement locking if it will be used
// by multiple goroutines.
type PubSub struct {
	log   logger.Logger
	clock clock.Clock

	eventsChan chan PubTrackEvent

	events *events

	// publishers is a map of publishers indexed by clientID of transport that
	// published the track and the track SSRC.
	publishers map[identifiers.TrackID]publisher

	// publishersByPubClientID is a map of a set of pubs that have been created by a
	// particular transport (indexes by clientID).
	publishersByPubClientID map[identifiers.ClientID]readerSet

	// subsBySubClientID is a map of a set of publishers that the transport has
	// subscribed to.
	subsBySubClientID map[identifiers.ClientID]subscriber
}

type publisher struct {
	clientID         identifiers.ClientID
	reader           Reader
	bitrateEstimator *BitrateEstimator
	timestamp        time.Time
}

type subscriber struct {
	transport         Transport
	publishersByTrack map[identifiers.TrackID]publisher
}

// New returns a new instance of PubSub.
func New(log logger.Logger, cl clock.Clock) *PubSub {
	eventsChan := make(chan PubTrackEvent)

	return &PubSub{
		log:                     log.WithNamespaceAppended("pubsub"),
		clock:                   cl,
		eventsChan:              eventsChan,
		events:                  newEvents(eventsChan, 0),
		publishers:              map[identifiers.TrackID]publisher{},
		publishersByPubClientID: map[identifiers.ClientID]readerSet{},
		subsBySubClientID:       map[identifiers.ClientID]subscriber{},
	}
}

// Pub publishes a track.
func (p *PubSub) Pub(pubClientID identifiers.ClientID, reader Reader) {
	track := reader.Track()

	p.log.Info("Pub", logger.Ctx{
		"client_id": pubClientID,
		"track_id":  track.TrackID(),
		"mime_type": track.Codec().MimeType,
	})

	trackID := track.TrackID()

	p.publishers[trackID] = publisher{
		clientID:         pubClientID,
		reader:           reader,
		bitrateEstimator: NewBitrateEstimator(),
		timestamp:        p.clock.Now(),
	}

	if _, ok := p.publishersByPubClientID[pubClientID]; !ok {
		p.publishersByPubClientID[pubClientID] = readerSet{}
	}

	p.publishersByPubClientID[pubClientID][reader] = struct{}{}

	prometheusWebRTCTracksTotal.Inc()
	prometheusWebRTCTracksActive.Inc()

	p.eventsChan <- PubTrackEvent{
		PubTrack: newPubTrack(pubClientID, track),
		Type:     transport.TrackEventTypeAdd,
	}
}

// Unpub unpublishes a track as well as unsubs all subscribers.
func (p *PubSub) Unpub(pubClientID identifiers.ClientID, trackID identifiers.TrackID) {
	p.log.Info("Unpub", logger.Ctx{
		"client_id": pubClientID,
		"track_id":  trackID,
	})

	if pub, ok := p.publishers[trackID]; ok {
		for _, subClientID := range pub.reader.Subs() {
			_ = p.unsub(subClientID, pub)
		}

		delete(p.publishersByPubClientID[pubClientID], pub.reader)

		if len(p.publishersByPubClientID[pubClientID]) == 0 {
			delete(p.publishersByPubClientID, pubClientID)
		}

		delete(p.publishers, trackID)

		prometheusWebRTCTracksActive.Dec()

		prometheusWebRTCTracksDuration.Observe(
			p.clock.Since(pub.timestamp).Seconds(),
		)

		p.eventsChan <- PubTrackEvent{
			PubTrack: newPubTrack(pubClientID, pub.reader.Track()),
			Type:     transport.TrackEventTypeRemove,
		}
	}
}

// Sub subscribes to a published track.
func (p *PubSub) Sub(pubClientID identifiers.ClientID, trackID identifiers.TrackID, transport Transport) (transport.RTCPReader, error) {
	p.log.Info("Sub", logger.Ctx{
		"client_id":     transport.ClientID(),
		"track_id":      trackID,
		"pub_client_id": pubClientID,
	})

	if pubClientID == transport.ClientID() {
		return nil, errors.Annotatef(ErrSubscribeToOwnTrack, "sub: trackID: %s, clientID: %s", trackID, transport.ClientID())
	}

	var err error

	pub, ok := p.publishers[trackID]
	if !ok {
		return nil, errors.Annotatef(ErrTrackNotFound, "sub: trackID: %s, clientID: %s", trackID, transport.ClientID())
	}

	sender, err := p.sub(pub, transport)
	if err != nil {
		return nil, errors.Annotatef(err, "sub: trackID: %s, clientID: %s", trackID, transport.ClientID())
	}

	return sender, nil
}

func (p *PubSub) sub(pub publisher, tr Transport) (transport.RTCPReader, error) {
	subClientID := tr.ClientID()

	track := pub.reader.Track()

	trackLocal, rtcpReader, err := tr.AddTrack(track)
	if err != nil {
		return nil, errors.Annotatef(err, "adding track to transport")
	}

	if err := pub.reader.Sub(subClientID, trackLocal); err != nil {
		// We don't care about the potential error at this point.
		_ = tr.RemoveTrack(track.TrackID())
		// TODO what to do with the track now?
		return nil, errors.Trace(err)
	}

	if _, ok := p.subsBySubClientID[subClientID]; !ok {
		p.subsBySubClientID[subClientID] = subscriber{
			transport:         tr,
			publishersByTrack: map[identifiers.TrackID]publisher{},
		}
	}

	p.subsBySubClientID[subClientID].publishersByTrack[track.TrackID()] = pub

	return rtcpReader, nil
}

// Unsub unsubscribes from a published track.
func (p *PubSub) Unsub(pubClientID identifiers.ClientID, trackID identifiers.TrackID, subClientID identifiers.ClientID) error {
	p.log.Info("Unsub", logger.Ctx{
		"client_id":     subClientID,
		"track_id":      trackID,
		"pub_client_id": pubClientID,
	})

	var err error

	pub, ok := p.publishers[trackID]
	if !ok {
		return errors.Annotatef(ErrTrackNotFound, "unsub: trackID: %s, clientID: %s", trackID, subClientID)
	}

	if err := p.unsub(subClientID, pub); err != nil {
		return errors.Annotatef(err, "unsub: trackiD: %s, clientID: %s", trackID, subClientID)
	}

	return errors.Trace(err)
}

// unsub caller must hold the lock.
func (p *PubSub) unsub(subClientID identifiers.ClientID, pub publisher) error {
	var multiErr multierr.MultiErr

	trackID := pub.reader.Track().TrackID()

	pub.bitrateEstimator.RemoveClientBitrate(subClientID)

	err := pub.reader.Unsub(subClientID)
	multiErr.Add(errors.Trace(err))

	sub, ok := p.subsBySubClientID[subClientID]
	if !ok {
		return errors.Annotatef(ErrSubNotFound, "subscriber not found")
	}

	err = sub.transport.RemoveTrack(trackID)
	multiErr.Add(errors.Trace(err))

	delete(p.subsBySubClientID[subClientID].publishersByTrack, trackID)

	if len(p.subsBySubClientID[subClientID].publishersByTrack) == 0 {
		delete(p.subsBySubClientID, subClientID)
	}

	return errors.Trace(multiErr.Err())
}

// BitrateEstimator returns the instance of BitrateEstimatro for a track.
func (p *PubSub) BitrateEstimator(trackID identifiers.TrackID) (*BitrateEstimator, bool) {
	pub, ok := p.publishers[trackID]

	return pub.bitrateEstimator, ok
}

// Terminate unpublishes al tracks from from a particular client, as well as
// removes any subscriptions it has.
func (p *PubSub) Terminate(clientID identifiers.ClientID) {
	p.log.Trace("Terminate", logger.Ctx{
		"client_id": clientID,
	})

	for reader := range p.publishersByPubClientID[clientID] {
		p.Unpub(clientID, reader.Track().TrackID())
	}

	for _, reader := range p.subsBySubClientID[clientID].publishersByTrack {
		_ = p.unsub(clientID, reader)
	}
}

// Subscribers returns all subscribed subClientIDs to a specific clientID/track
// pair.
func (p *PubSub) Subscribers(pubClientID identifiers.ClientID, trackID identifiers.TrackID) []identifiers.ClientID {
	var ret []identifiers.ClientID

	if pub, ok := p.publishers[trackID]; ok {
		subs := pub.reader.Subs()

		if l := len(subs); l > 0 {
			ret = make([]identifiers.ClientID, l)

			copy(ret, subs)
		}
	}

	return ret
}

type TrackProps struct {
	ClientID identifiers.ClientID
	SSRC     webrtc.SSRC
	RID      string
}

// ClientIDByTrackID returns the clientID from a published unique trackID.
func (p *PubSub) TrackPropsByTrackID(trackID identifiers.TrackID) (TrackProps, bool) {
	pub, ok := p.publishers[trackID]

	if !ok {
		return TrackProps{}, false
	}

	return TrackProps{
		ClientID: pub.clientID,
		SSRC:     pub.reader.SSRC(),
		RID:      pub.reader.RID(),
	}, true
}

// Tracks returns all published track information. The order is undefined.
func (p *PubSub) Tracks() []PubTrack {
	var ret []PubTrack

	if l := len(p.publishers); l > 0 {
		ret = make([]PubTrack, 0, l)

		for _, pub := range p.publishers {
			ret = append(ret, newPubTrack(pub.clientID, pub.reader.Track()))
		}
	}

	p.log.Trace(fmt.Sprintf("Tracks: %d", len(ret)), nil)

	return ret
}

// SubscribeToEvents creates a new subscription to track events.
func (p *PubSub) SubscribeToEvents(clientID identifiers.ClientID) (<-chan PubTrackEvent, error) {
	p.log.Trace("SubscribeToEvents", logger.Ctx{
		"client_id": clientID,
	})

	ch, err := p.events.Subscribe(clientID)

	return ch, errors.Annotatef(err, "sub events: clientID: %s", clientID)
}

// UnsubscribeFromEvents removes an existing subscription from track events.
func (p *PubSub) UnsubscribeFromEvents(clientID identifiers.ClientID) error {
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
