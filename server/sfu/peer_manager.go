package sfu

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/pubsub"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type PeerManager struct {
	log logger.Logger
	mu  sync.RWMutex
	wg  sync.WaitGroup

	jitterHandler          JitterHandler
	trackBitrateEstimators *TrackBitrateEstimators

	// webrtcTransports indexed by ClientID
	webrtcTransports map[string]transport.Transport
	serverTransports map[string]transport.Transport

	room string

	// pubsub keeps track of published tracks and its subscribers.
	pubsub *pubsub.PubSub
}

func NewPeerManager(room string, log logger.Logger, jitterHandler JitterHandler) *PeerManager {
	return &PeerManager{
		log:                    log.WithNamespaceAppended("room_peers_manager"),
		webrtcTransports:       map[string]transport.Transport{},
		serverTransports:       map[string]transport.Transport{},
		jitterHandler:          jitterHandler,
		trackBitrateEstimators: NewTrackBitrateEstimators(),
		room:                   room,

		pubsub: pubsub.New(),
	}
}

func (t *PeerManager) addTrack(clientID string, track transport.Track) {
	t.mu.Lock()
	defer t.mu.Unlock()

	log := t.log.WithCtx(logger.Ctx{
		"client_id": clientID,
		"ssrc":      track.SSRC(),
	})

	log.Trace("Add track", logger.Ctx{
		"track": track,
	})

	t.pubsub.Pub(clientID, track)

	// Let the server transports know of the new track.
	for subClientID, subTransport := range t.serverTransports {
		if subClientID != clientID {
			if err := t.pubsub.Sub(clientID, track.SSRC(), subTransport); err != nil {
				log.Error("Add track", errors.Trace(err), logger.Ctx{
					"sub_client_id": subClientID,
				})

				continue
			}
		}
	}
}

func (t *PeerManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for otherClientID, otherPeerInRoom := range t.webrtcTransports {
		if otherClientID != clientID {
			// FIXME async
			err := <-otherPeerInRoom.Send(msg)
			if err != nil {
				t.log.Error("Broadcast", errors.Trace(err), logger.Ctx{
					"client_id":       clientID,
					"other_client_id": otherClientID,
				})
			}
		}
	}
}

func (t *PeerManager) getTransportBySSRC(subClientID string, ssrc uint32) (
	transport transport.Transport, ok bool,
) {
	t.mu.Lock()
	defer t.mu.Unlock()

	clientID, ok := t.pubsub.PubClientID(subClientID, ssrc)
	if !ok {
		return nil, false
	}

	transport, ok = t.webrtcTransports[clientID]

	return transport, ok
}

func (t *PeerManager) Add(tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	log := t.log.WithCtx(logger.Ctx{
		"client_id": tr.ClientID(),
	})

	pubTrackEventSub, err := t.pubsub.SubscribeToEvents(tr.ClientID())
	if err != nil {
		return nil, errors.Annotatef(err, "subscribe to events: %s", tr.ClientID())
	}

	pubTracks := t.pubsub.Tracks()

	pubTrackEventsCh := make(chan pubsub.PubTrackEvent)

	t.wg.Add(1)

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		defer close(pubTrackEventsCh)

		for _, pubTrack := range pubTracks {
			if pubTrack.ClientID != tr.ClientID() {
				pubTrackEventsCh <- pubsub.PubTrackEvent{
					PubTrack: pubsub.PubTrack{
						ClientID: pubTrack.ClientID,
						UserID:   pubTrack.UserID,
						SSRC:     pubTrack.SSRC,
					},
					Type: transport.TrackEventTypeAdd,
				}
			}
		}

		for event := range pubTrackEventSub {
			if event.PubTrack.ClientID != tr.ClientID() {
				pubTrackEventsCh <- event
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for trackEvent := range tr.TrackEventsChannel() {
			switch trackEvent.Type {
			case transport.TrackEventTypeAdd:
				t.addTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
			case transport.TrackEventTypeRemove:
				t.removeTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
				// The following events are generated only by server transport.
			case transport.TrackEventTypeSub:
				if err := t.Sub(SubParams{
					Room:        t.room,
					PubClientID: trackEvent.ClientID,
					SSRC:        trackEvent.TrackInfo.Track.SSRC(),
					SubClientID: tr.ClientID(),
				}); err != nil {
					log.Error("sub failed", err, nil)
				}
			case transport.TrackEventTypeUnsub:
				if err := t.Unsub(SubParams{
					Room:        t.room,
					PubClientID: trackEvent.ClientID,
					SSRC:        trackEvent.TrackInfo.Track.SSRC(),
					SubClientID: tr.ClientID(),
				}); err != nil {
					log.Error("sub failed", err, nil)
				}
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for packet := range tr.RTPChannel() {
			rtcpPacket := t.jitterHandler.HandleRTP(packet)
			if rtcpPacket != nil {
				err := tr.WriteRTCP([]rtcp.Packet{rtcpPacket})
				if err != nil {
					log.Error("WriteRTCP", errors.Trace(err), nil)
				}
			}

			t.mu.Lock()

			subTransports := t.pubsub.Subscribers(tr.ClientID(), packet.SSRC)

			t.mu.Unlock()

			for subClientID, subTransport := range subTransports {
				if _, err := subTransport.(transport.Transport).WriteRTP(packet); err != nil {
					log.Error("WriteRTP", errors.Trace(err), logger.Ctx{
						"pub_client_id": tr.ClientID(),
						"sub_client_id": subClientID,
						"ssrc":          packet.SSRC,
					})
				}
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		handleREMB := func(packet *rtcp.ReceiverEstimatedMaximumBitrate) error {
			errs := multierr.New()

			bitrate := t.trackBitrateEstimators.Estimate(tr.ClientID(), packet.SSRCs, packet.Bitrate)
			packet.Bitrate = bitrate

			transportsSet := map[transport.Transport]struct{}{}

			for _, ssrc := range packet.SSRCs {
				sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), ssrc)
				if ok {
					transportsSet[sourceTransport] = struct{}{}
				}
			}

			for sourceTransport := range transportsSet {
				err := sourceTransport.WriteRTCP([]rtcp.Packet{packet})
				errs.Add(errors.Trace(err))
			}

			return errors.Annotatef(errs.Err(), "remb")
		}

		handlePLI := func(packet *rtcp.PictureLossIndication) error {
			sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), packet.MediaSSRC)
			if !ok {
				return errors.Errorf("no source transport for PictureLossIndication for track: %d", packet.MediaSSRC)
			}

			err := sourceTransport.WriteRTCP([]rtcp.Packet{packet})

			return errors.Annotate(err, "write rtcp")
		}

		handleNack := func(packet *rtcp.TransportLayerNack) error {
			errs := multierr.New()

			foundRTPPackets, nack := t.jitterHandler.HandleNack(packet)
			for _, rtpPacket := range foundRTPPackets {
				if _, err := tr.WriteRTP(rtpPacket); err != nil {
					errs.Add(errors.Annotate(err, "write rtp"))
				}
			}

			if nack != nil {
				sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), packet.MediaSSRC)
				if ok {
					if err := sourceTransport.WriteRTCP([]rtcp.Packet{nack}); err != nil {
						errs.Add(errors.Annotate(err, "write rtcp"))
					}
				}
			}

			return errors.Annotatef(errs.Err(), "nack")
		}

		for pkt := range tr.RTCPChannel() {
			var err error
			switch packet := pkt.(type) {
			case *rtcp.ReceiverEstimatedMaximumBitrate:
				err = errors.Trace(handleREMB(packet))
			case *rtcp.PictureLossIndication:
				err = errors.Trace(handlePLI(packet))
			case *rtcp.TransportLayerNack:
				err = errors.Trace(handleNack(packet))
			case *rtcp.SourceDescription:
			case *rtcp.ReceiverReport:
			case *rtcp.SenderReport:
			default:
				t.log.Error("Unhandled RTCP Packet", nil, logger.Ctx{
					"destination_ssrc": pkt.DestinationSSRC(),
				})
			}

			if err != nil {
				// Log error and do not return early because the RTCP channel still
				// needs to be emptied.
				t.log.Error("Send RTCP to source peer", errors.Trace(err), nil)
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for msg := range tr.MessagesChannel() {
			t.broadcast(tr.ClientID(), msg)
		}
	}()

	t.wg.Done()

	t.mu.Lock()
	defer t.mu.Unlock()

	switch tr.Type() {
	case transport.TypeServer:
		t.serverTransports[tr.ClientID()] = tr

		for _, pubTransport := range t.webrtcTransports {
			for _, trackInfo := range pubTransport.RemoteTracks() {
				if err := pubTransport.AddTrack(trackInfo.Track); err != nil {
					t.log.Error("add track", errors.Trace(err), logger.Ctx{
						"pub_client_id": pubTransport.ClientID(),
						"sub_client_id": tr.ClientID(),
						"ssrc":          trackInfo.Track.SSRC(),
					})
				}
			}
		}

	case transport.TypeWebRTC:
		t.webrtcTransports[tr.ClientID()] = tr
	}

	return pubTrackEventsCh, nil
}

func (t *PeerManager) Sub(params SubParams) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	transport, ok := t.webrtcTransports[params.SubClientID]
	if !ok {
		return errors.Errorf("transport not found: %s", params.PubClientID)
	}

	err := t.pubsub.Sub(params.PubClientID, params.SSRC, transport)

	return errors.Trace(err)
}

func (t *PeerManager) Unsub(params SubParams) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.pubsub.Unsub(params.PubClientID, params.SSRC, params.SubClientID)

	return errors.Trace(err)
}

// TracksMetadata retrieves local track metadata for a specific transport.
func (t *PeerManager) TracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	transport, ok := t.webrtcTransports[clientID]
	if !ok {
		return m, false
	}

	tracks := transport.LocalTracks()
	m = make([]TrackMetadata, 0, len(tracks))

	for _, trackInfo := range tracks {
		track := trackInfo.Track.(UserTrack)

		trackMetadata := TrackMetadata{
			Kind:     trackInfo.Kind.String(),
			Mid:      trackInfo.Mid,
			StreamID: track.Label(),
			UserID:   track.UserID(),
		}

		t.log.Trace("GetTracksMetadata", logger.Ctx{
			"ssrc":      track.SSRC(),
			"client_id": clientID,
		})

		m = append(m, trackMetadata)
	}

	return m, true
}

func (t *PeerManager) Remove(clientID string) {
	t.log.Trace("Remove", logger.Ctx{
		"client_id": clientID,
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.pubsub.UnsubscribeFromEvents(clientID); err != nil {
		t.log.Error("Unsubscribe from events", errors.Trace(err), logger.Ctx{
			"client_id": clientID,
		})
	}

	t.trackBitrateEstimators.RemoveReceiverEstimations(clientID)
	delete(t.webrtcTransports, clientID)
}

func (t *PeerManager) removeTrack(clientID string, track transport.Track) {
	logCtx := logger.Ctx{
		"client_id": clientID,
		"ssrc":      track.SSRC(),
	}

	t.log.Trace("Remove track", logCtx)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.pubsub.Unpub(clientID, track.SSRC())

	t.trackBitrateEstimators.Remove(track.SSRC())

	// Let the server transports know the track has been removed
	for subClientID, subTransport := range t.serverTransports {
		if subClientID != clientID {
			if err := subTransport.RemoveTrack(track.SSRC()); err != nil {
				t.log.Error("Remove track", errors.Trace(err), logger.Ctx{
					"sub_client_id": subClientID,
					"ssrc":          track.SSRC(),
				})

				continue
			}
		}
	}
}

func (t *PeerManager) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.webrtcTransports)
}

func (t *PeerManager) Close() <-chan struct{} {
	ch := make(chan struct{}, 1)

	go func() {
		t.wg.Wait()
		t.pubsub.Close()

		close(ch)
	}()

	return ch
}
