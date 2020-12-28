package server

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

const DataChannelName = "data"

type TrackMetadata struct {
	Mid      string `json:"mid"`
	UserID   string `json:"userId"`
	StreamID string `json:"streamId"`
	Kind     string `json:"kind"`
}

type MemoryTracksManager struct {
	log                 logger.Logger
	mu                  sync.RWMutex
	roomPeersManager    map[string]*RoomPeersManager
	jitterBufferEnabled bool
}

func NewMemoryTracksManager(log logger.Logger, jitterBufferEnabled bool) *MemoryTracksManager {
	return &MemoryTracksManager{
		log:                 log.WithNamespaceAppended("memory_tracks_manager"),
		roomPeersManager:    map[string]*RoomPeersManager{},
		jitterBufferEnabled: jitterBufferEnabled,
	}
}

func (m *MemoryTracksManager) Add(room string, transport Transport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := m.log.WithCtx(logger.Ctx{
		"room_id": room,
	})

	roomPeersManager, ok := m.roomPeersManager[room]
	if !ok {
		jitterHandler := NewJitterHandler(
			log,
			m.jitterBufferEnabled,
		)
		roomPeersManager = NewRoomPeersManager(room, m.log, jitterHandler)
		m.roomPeersManager[room] = roomPeersManager

		// TODO Write to RoomEventsChan
	}

	log = log.WithCtx(logger.Ctx{
		"client_id": transport.ClientID(),
	})

	log.Info("Add peer", nil)
	roomPeersManager.Add(transport)

	go func() {
		<-transport.CloseChannel()
		m.mu.Lock()
		defer m.mu.Unlock()

		roomPeersManager.Remove(transport.ClientID())

		// TODO tell the difference between server and webrtc transports since
		// server transports should not be counted, and they should be removed.
		if len(roomPeersManager.transports) == 0 {
			// TODO write to RoomEventsChan
			delete(m.roomPeersManager, room)
		}
	}()
}

func (m *MemoryTracksManager) GetTracksMetadata(room string, clientID string) (metadata []TrackMetadata, ok bool) {
	m.mu.RLock()
	roomPeersManager, ok := m.roomPeersManager[room]
	m.mu.RUnlock()

	if !ok {
		return metadata, false
	}

	return roomPeersManager.GetTracksMetadata(clientID)
}

func (m *MemoryTracksManager) Subscribe(params SubscribeParams) error {
	return errors.Errorf("Not implemented")
}

func (m *MemoryTracksManager) Unsubscribe(params SubscribeParams) error {
	return errors.Errorf("Not implemented")
}

type RoomPeersManager struct {
	log logger.Logger
	mu  sync.RWMutex
	// key is clientID
	transports             map[string]Transport
	jitterHandler          JitterHandler
	trackBitrateEstimators *TrackBitrateEstimators
	clientIDBySSRC         map[uint32]string
	room                   string
}

func NewRoomPeersManager(room string, log logger.Logger, jitterHandler JitterHandler) *RoomPeersManager {
	return &RoomPeersManager{
		log:                    log.WithNamespaceAppended("room_peers_manager"),
		transports:             map[string]Transport{},
		jitterHandler:          jitterHandler,
		trackBitrateEstimators: NewTrackBitrateEstimators(),
		clientIDBySSRC:         map[uint32]string{},
		room:                   room,
	}
}

func (t *RoomPeersManager) addTrack(clientID string, track Track) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.clientIDBySSRC[track.SSRC()] = clientID

	log := t.log.WithCtx(logger.Ctx{
		"client_id": clientID,
	})

	for otherClientID, otherTransport := range t.transports {
		if otherClientID != clientID {

			log.Trace("Add track (BEFORE)", logger.Ctx{
				"other_client_id": otherClientID,
				"ssrc":            track.SSRC(),
				"track":           track,
			})

			// TODO store the track associations in the map and let the clients
			// subscribe as needed instead of subscribing automatically.
			track := t.asUserTrack(track, clientID)

			log.Trace("Add track (AFTER)", logger.Ctx{
				"other_client_id": otherClientID,
				"ssrc":            track.SSRC(),
				"track":           track,
			})

			if err := otherTransport.AddTrack(track); err != nil {
				log.Error("Add track", errors.Trace(err), nil)
				continue
			}
		}
	}
}

func (t *RoomPeersManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
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

func (t *RoomPeersManager) getTransportBySSRC(ssrc uint32) (transport Transport, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	clientID, ok := t.clientIDBySSRC[ssrc]
	if !ok {
		return nil, false
	}

	transport, ok = t.transports[clientID]
	return transport, ok
}

func (t *RoomPeersManager) Add(transport Transport) {
	log := t.log.WithCtx(logger.Ctx{
		"client_id": transport.ClientID(),
	})

	go func() {
		for trackEvent := range transport.TrackEventsChannel() {
			switch trackEvent.Type {
			case TrackEventTypeAdd:
				t.addTrack(transport.ClientID(), trackEvent.TrackInfo.Track)
			case TrackEventTypeRemove:
				t.removeTrack(transport.ClientID(), trackEvent.TrackInfo.Track)
			}
		}
	}()

	go func() {
		for packet := range transport.RTPChannel() {
			rtcpPacket := t.jitterHandler.HandleRTP(packet)
			if rtcpPacket != nil {
				err := transport.WriteRTCP([]rtcp.Packet{rtcpPacket})
				if err != nil {
					log.Error("WriteRTCP", errors.Trace(err), nil)
				}
			}

			// FIXME Do not lock (block) on long operations such as WriteRTP below.
			t.mu.Lock()

			for otherClientID, otherTransport := range t.transports {
				if otherClientID != transport.ClientID() {
					_, err := otherTransport.WriteRTP(packet)
					if err != nil {
						log.Error("WriteRTP", errors.Trace(err), nil)
					}
				}
			}

			t.mu.Unlock()
		}
	}()

	go func() {
		handleREMB := func(packet *rtcp.ReceiverEstimatedMaximumBitrate) error {
			var errs MultiErrorHandler

			bitrate := t.trackBitrateEstimators.Estimate(transport.ClientID(), packet.SSRCs, packet.Bitrate)
			packet.Bitrate = bitrate

			transportsSet := map[Transport]struct{}{}

			for _, ssrc := range packet.SSRCs {
				sourceTransport, ok := t.getTransportBySSRC(ssrc)
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
			sourceTransport, ok := t.getTransportBySSRC(packet.MediaSSRC)
			if !ok {
				return errors.Errorf("no source transport for PictureLossIndication for track: %d", packet.MediaSSRC)
			}

			err := sourceTransport.WriteRTCP([]rtcp.Packet{packet})
			return errors.Annotate(err, "write rtcp")
		}

		handleNack := func(packet *rtcp.TransportLayerNack) error {
			var errs MultiErrorHandler

			foundRTPPackets, nack := t.jitterHandler.HandleNack(packet)
			for _, rtpPacket := range foundRTPPackets {
				if _, err := transport.WriteRTP(rtpPacket); err != nil {
					errs.Add(errors.Annotate(err, "write rtp"))
				}
			}

			if nack != nil {
				sourceTransport, ok := t.getTransportBySSRC(packet.MediaSSRC)
				if ok {
					if err := sourceTransport.WriteRTCP([]rtcp.Packet{nack}); err != nil {
						errs.Add(errors.Annotate(err, "write rtcp"))
					}
				}
			}

			return errors.Annotatef(errs.Err(), "nack")
		}

		for pkt := range transport.RTCPChannel() {
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
				t.log.Error("Send RTCP to source peer", errors.Trace(err), nil)
				// do not return early since the rtcp channel needs to be emptied
			}
		}
	}()

	go func() {
		for msg := range transport.MessagesChannel() {
			t.broadcast(transport.ClientID(), msg)
		}
	}()

	t.mu.Lock()
	defer t.mu.Unlock()

	for existingClientID, existingTransport := range t.transports {
		for _, trackInfo := range existingTransport.RemoteTracks() {

			log.Trace("Add track to existing (BEFORE)", logger.Ctx{
				"other_client_id": existingClientID,
				"ssrc":            trackInfo.Track.SSRC(),
				"track":           trackInfo.Track,
			})

			track := t.asUserTrack(trackInfo.Track, existingClientID)

			log.Trace("Add track to existing (AFTER)", logger.Ctx{
				"other_client_id": existingClientID,
				"ssrc":            track.SSRC(),
				"track":           track,
			})

			err := transport.AddTrack(track)
			if err != nil {
				err = errors.Annotatef(err, "add track")
				t.log.Error("Add track", errors.Trace(err), logger.Ctx{
					"other_client_id": existingClientID,
				})
			}
		}
	}

	t.transports[transport.ClientID()] = transport
}

// getUserID tries to obtain to userID from a track, but otherwise falls back
// to the clientID.
func (t *RoomPeersManager) getUserID(track Track) string {
	var userID string
	if userIdentifiable, ok := track.(UserIdentifiable); ok {
		userID = userIdentifiable.UserID()
	}
	if userID == "" {
		userID = t.clientIDBySSRC[track.SSRC()]
	}
	return userID
}

// asUserTrack adds business level metadata to track such as userID and roomID
// if such data does not already exist.
func (t *RoomPeersManager) asUserTrack(track Track, clientID string) Track {
	if _, ok := track.(UserIdentifiable); ok {
		return track
	}

	return NewUserTrack(track, clientID, t.room)
}

// GetTracksMetadata retrieves remote track metadata for a specific peer.
//
// TODO In the future, this method will need to be implemented differently for
// WebRTCTransport and RTPTransport, since RTPTransport might contain tracks
// for multiple users in a peer. Therefor the RTPTransport should be able to
// provide metadata on its own.
func (t *RoomPeersManager) GetTracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	transport, ok := t.transports[clientID]
	if !ok {
		return m, false
	}

	tracks := transport.LocalTracks()
	m = make([]TrackMetadata, 0, len(tracks))

	for _, trackInfo := range tracks {
		track := trackInfo.Track

		trackMetadata := TrackMetadata{
			Kind:     trackInfo.Kind.String(),
			Mid:      trackInfo.Mid,
			StreamID: track.Label(),
			UserID:   t.getUserID(track),
		}

		t.log.Trace("GetTracksMetadata", logger.Ctx{
			"ssrc":      track.SSRC(),
			"client_id": clientID,
		})

		m = append(m, trackMetadata)
	}

	return m, true
}

func (t *RoomPeersManager) Remove(clientID string) {
	t.log.Trace("Remove", logger.Ctx{
		"client_id": clientID,
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	t.trackBitrateEstimators.RemoveReceiverEstimations(clientID)
	delete(t.transports, clientID)
}

func (t *RoomPeersManager) removeTrack(clientID string, track Track) {
	logCtx := logger.Ctx{
		"client_id": clientID,
		"ssrc":      track.SSRC(),
	}

	t.log.Trace("Remove track", logCtx)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.trackBitrateEstimators.Remove(track.SSRC())
	delete(t.clientIDBySSRC, track.SSRC())

	for otherClientID, otherTransport := range t.transports {
		if otherClientID != clientID {
			err := otherTransport.RemoveTrack(track.SSRC())
			if err != nil {
				t.log.Error("Remove track", errors.Trace(err), logger.Ctx{
					"other_client_id": otherClientID,
				})
			}
		}
	}
}
