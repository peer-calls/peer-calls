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
	log                    logger.Logger
	mu                     sync.RWMutex
	jitterHandler          JitterHandler
	trackBitrateEstimators *TrackBitrateEstimators

	// transports indexed by ClientID
	transports map[string]transport.Transport
	room       string

	// pubsub keeps track of published tracks and its subscribers.
	pubsub *pubsub.PubSub
}

func NewPeerManager(room string, log logger.Logger, jitterHandler JitterHandler) *PeerManager {
	return &PeerManager{
		log:                    log.WithNamespaceAppended("room_peers_manager"),
		transports:             map[string]transport.Transport{},
		jitterHandler:          jitterHandler,
		trackBitrateEstimators: NewTrackBitrateEstimators(),
		room:                   room,

		pubsub: pubsub.New(),
		// trackEventsSuber: newTrackEventsSuber(),
	}
}

func (t *PeerManager) addTrack(clientID string, track transport.Track) {
	t.mu.Lock()
	defer t.mu.Unlock()

	log := t.log.WithCtx(logger.Ctx{
		"client_id": clientID,
		"ssrc":      track.SSRC(),
	})

	log.Trace("Add track (BEFORE)", logger.Ctx{
		"track": track,
	})

	track = t.asUserTrack(track, clientID)

	log.Trace("Add track (AFTER)", logger.Ctx{
		"track": track,
	})

	t.pubsub.Pub(clientID, track)

	// TODO store the track associations in the map and let the clients
	// subscribe as needed instead of subscribing automatically.
	for subClientID, subTransport := range t.transports {
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

	for otherClientID, otherPeerInRoom := range t.transports {
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

	transport, ok = t.transports[clientID]

	return transport, ok
}

func (t *PeerManager) Add(tr transport.Transport) {
	log := t.log.WithCtx(logger.Ctx{
		"client_id": tr.ClientID(),
	})

	go func() {
		for trackEvent := range tr.TrackEventsChannel() {
			switch trackEvent.Type {
			case transport.TrackEventTypeAdd:
				t.addTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
			case transport.TrackEventTypeRemove:
				t.removeTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
			}
		}
	}()

	go func() {
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

	go func() {
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

	go func() {
		for msg := range tr.MessagesChannel() {
			t.broadcast(tr.ClientID(), msg)
		}
	}()

	t.mu.Lock()
	defer t.mu.Unlock()

	for pubClientID, pubTransport := range t.transports {
		for _, trackInfo := range pubTransport.RemoteTracks() {
			if err := t.pubsub.Sub(pubClientID, trackInfo.Track.SSRC(), tr); err != nil {
				err = errors.Annotatef(err, "add track")
				t.log.Error("sub", errors.Trace(err), logger.Ctx{
					"pub_client_id": pubTransport.ClientID(),
					"sub_client_id": tr.ClientID(),
					"ssrc":          trackInfo.Track.SSRC(),
				})
			}
		}
	}

	t.transports[tr.ClientID()] = tr
}

func (t *PeerManager) Sub(params SubParams) error {
	transport, ok := t.transports[params.SubClientID]
	if !ok {
		return errors.Errorf("transport not found: %s", params.PubClientID)
	}

	err := t.pubsub.Sub(params.PubClientID, params.SSRC, transport)

	return errors.Trace(err)
}

func (t *PeerManager) Unsub(params SubParams) error {
	err := t.pubsub.Unsub(params.PubClientID, params.SSRC, params.SubClientID)

	return errors.Trace(err)
}

// asUserTrack adds business level metadata to track such as userID and roomID
// if such data does not already exist.
func (t *PeerManager) asUserTrack(track transport.Track, clientID string) transport.Track {
	if _, ok := track.(userIdentifiable); ok {
		return track
	}

	t.log.Warn("Unexpected non-user track", logger.Ctx{
		"track":     track,
		"ssrc":      track.SSRC(),
		"client_id": clientID,
	})

	return NewUserTrack(track, clientID, t.room)
}

// GetTracksMetadata retrieves local track metadata for a specific peer.
//
// TODO In the future, this method will need to be implemented differently for
// WebRTCTransport and RTPTransport, since RTPTransport might contain tracks
// for multiple users in a peer. Therefor the RTPTransport should be able to
// provide metadata on its own.
func (t *PeerManager) GetTracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	transport, ok := t.transports[clientID]
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

	t.trackBitrateEstimators.RemoveReceiverEstimations(clientID)
	delete(t.transports, clientID)
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
}

func (t *PeerManager) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.transports)
}

type userIdentifiable interface {
	UserID() string
}
