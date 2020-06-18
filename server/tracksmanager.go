package server

import (
	"fmt"
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
)

const DataChannelName = "data"

type TrackMetadata struct {
	Mid      string `json:"mid"`
	UserID   string `json:"userId"`
	StreamID string `json:"streamId"`
	Kind     string `json:"kind"`
}

type MemoryTracksManager struct {
	loggerFactory       LoggerFactory
	log                 Logger
	mu                  sync.RWMutex
	roomPeersManager    map[string]*RoomPeersManager
	jitterBufferEnabled bool
}

func NewMemoryTracksManager(loggerFactory LoggerFactory, jitterBufferEnabled bool) *MemoryTracksManager {
	return &MemoryTracksManager{
		loggerFactory:       loggerFactory,
		log:                 loggerFactory.GetLogger("memorytracksmanager"),
		roomPeersManager:    map[string]*RoomPeersManager{},
		jitterBufferEnabled: jitterBufferEnabled,
	}
}

func (m *MemoryTracksManager) Add(room string, transport Transport) {

	m.mu.Lock()
	defer m.mu.Unlock()

	roomPeersManager, ok := m.roomPeersManager[room]
	if !ok {
		jitterHandler := NewJitterHandler(
			m.loggerFactory.GetLogger("jitter"),
			m.loggerFactory.GetLogger("nack"),
			m.jitterBufferEnabled,
		)
		roomPeersManager = NewRoomPeersManager(room, m.loggerFactory, jitterHandler)
		m.roomPeersManager[room] = roomPeersManager

		// TODO Write to RoomEventsChan
	}

	m.log.Printf("[%s] MemoryTrackManager.Add peer to room: %s", transport.ClientID(), room)
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
	roomPeersManager, ok := m.roomPeersManager[room]
	if !ok {
		return metadata, false
	}
	return roomPeersManager.GetTracksMetadata(clientID)
}

type RoomPeersManager struct {
	loggerFactory LoggerFactory
	log           Logger
	mu            sync.RWMutex
	// key is clientID
	transports             map[string]Transport
	jitterHandler          JitterHandler
	trackBitrateEstimators *TrackBitrateEstimators
	clientIDBySSRC         map[uint32]string
	room                   string
}

func NewRoomPeersManager(room string, loggerFactory LoggerFactory, jitterHandler JitterHandler) *RoomPeersManager {
	return &RoomPeersManager{
		loggerFactory:          loggerFactory,
		log:                    loggerFactory.GetLogger("roompeers"),
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

	for otherClientID, otherTransport := range t.transports {
		if otherClientID != clientID {
			track := t.asUserTrack(track, otherClientID)
			if err := otherTransport.AddTrack(track); err != nil {
				t.log.Printf("[%s] MemoryTracksManager.addTrack Error adding track: %s", otherClientID, err)
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
			t.log.Printf("[%s] broadcast from %s", otherClientID, clientID)
			var err error
			if msg.IsString {
				err = otherPeerInRoom.SendText(string(msg.Data))
			} else {
				err = otherPeerInRoom.Send(msg.Data)
			}
			if err != nil {
				t.log.Printf("[%s] broadcast error: %s", otherClientID, err)
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
	go func() {
		for trackEvent := range transport.TrackEventsChannel() {
			switch trackEvent.Type {
			case TrackEventTypeAdd:
				t.addTrack(transport.ClientID(), trackEvent.Track)
			case TrackEventTypeRemove:
				t.removeTrack(transport.ClientID(), trackEvent.Track)
			}
		}
	}()

	go func() {
		for packet := range transport.RTPChannel() {
			rtcpPacket := t.jitterHandler.HandleRTP(packet)
			if rtcpPacket != nil {
				err := transport.WriteRTCP([]rtcp.Packet{rtcpPacket})
				if err != nil {
					t.log.Printf("[%s] Error writing RTCP packet: %s: %s", transport.ClientID(), rtcpPacket, err)
				}
			}

			t.mu.Lock()

			for otherClientID, otherTransport := range t.transports {
				if otherClientID != transport.ClientID() {
					_, err := otherTransport.WriteRTP(packet)
					if err != nil {
						t.log.Printf("[%s] Error writing RTP packet for ssrc: %d: %s", otherClientID, packet.SSRC, err)
					}
				}
			}

			t.mu.Unlock()
		}
	}()

	go func() {
		for pkt := range transport.RTCPChannel() {
			var err error
			switch packet := pkt.(type) {
			case *rtcp.ReceiverEstimatedMaximumBitrate:
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
					rtcpErr := sourceTransport.WriteRTCP([]rtcp.Packet{pkt})
					if err == nil && rtcpErr != nil {
						err = rtcpErr
					}
				}
			case *rtcp.PictureLossIndication:
				sourceTransport, ok := t.getTransportBySSRC(packet.MediaSSRC)
				if ok {
					err = sourceTransport.WriteRTCP([]rtcp.Packet{pkt})
				} else {
					err = fmt.Errorf("Cannot find source transport for PictureLossIndication for track: %d", packet.MediaSSRC)
				}
			case *rtcp.TransportLayerNack:
				foundRTPPackets, nack := t.jitterHandler.HandleNack(packet)
				for _, rtpPacket := range foundRTPPackets {
					_, err := transport.WriteRTP(rtpPacket)
					if err != nil {
						t.log.Printf("[%s] Error writing found RTP packet per NACK request for track: %d: %s", transport.ClientID(), rtpPacket.SSRC, err)
					} else {
						err = fmt.Errorf("Cannot find source transport for NACK for track: %d", packet.MediaSSRC)
					}
				}
				if nack != nil {
					sourceTransport, ok := t.getTransportBySSRC(packet.MediaSSRC)
					if ok {
						err = sourceTransport.WriteRTCP([]rtcp.Packet{nack})
					}
				}
			case *rtcp.SourceDescription:
			case *rtcp.ReceiverReport:
			case *rtcp.SenderReport:
			default:
				t.log.Printf("[%s] Got unhandled RTCP pkt for track: %d (%T)", transport.ClientID(), pkt.DestinationSSRC(), pkt)
			}
			if err != nil {
				t.log.Printf("[%s] addTrackToPeer error sending RTCP packet to source peer: %s", transport.ClientID(), err)
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
			track := t.asUserTrack(trackInfo.Track, existingClientID)
			err := transport.AddTrack(track)
			if err != nil {
				t.log.Printf(
					"Error adding peer clientID: %s track to clientID: %s - reason: %s",
					existingClientID,
					transport.ClientID(),
					err,
				)
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
		t.log.Printf("[%s] GetTracksMetadata: %d %#v", clientID, track.SSRC(), trackMetadata)
		m = append(m, trackMetadata)
	}

	return m, true
}

func (t *RoomPeersManager) Remove(clientID string) {
	t.log.Printf("removePeer: %s", clientID)
	t.mu.Lock()
	defer t.mu.Unlock()

	t.trackBitrateEstimators.RemoveReceiverEstimations(clientID)
	delete(t.transports, clientID)
}

func (t *RoomPeersManager) removeTrack(clientID string, track Track) {
	t.log.Printf("[%s] removeTrack ssrc: %d from other peers", clientID, track.SSRC())

	t.mu.Lock()
	defer t.mu.Unlock()

	t.trackBitrateEstimators.Remove(track.SSRC())
	delete(t.clientIDBySSRC, track.SSRC())

	for otherClientID, otherTransport := range t.transports {
		if otherClientID != clientID {
			err := otherTransport.RemoveTrack(track.SSRC())
			if err != nil {
				t.log.Printf("[%s] removeTrack error removing track: %s", clientID, err)
			}
		}
	}
}
