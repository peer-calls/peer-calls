package server

import (
	"fmt"
	"sync"

	"github.com/pion/webrtc/v2"
)

const DataChannelName = "data"

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

func (m *MemoryTracksManager) Add(
	room string,
	clientID string,
	peerConnection *webrtc.PeerConnection,
	dataChannel *webrtc.DataChannel,
	signaller *Signaller,
) {

	m.mu.Lock()
	defer m.mu.Unlock()

	roomPeersManager, ok := m.roomPeersManager[room]
	if !ok {
		jitterHandler := NewJitterHandler(m.loggerFactory.GetLogger("jitter"), m.jitterBufferEnabled)
		roomPeersManager = NewRoomPeersManager(m.loggerFactory, jitterHandler)
		m.roomPeersManager[room] = roomPeersManager
	}

	m.log.Printf("[%s] MemoryTrackManager.Add peer to room: %s", clientID, room)
	roomPeersManager.Add(clientID, peerConnection, dataChannel, signaller)

	go func() {
		<-signaller.CloseChannel()
		m.mu.Lock()
		defer m.mu.Unlock()

		roomPeersManager.Remove(clientID)

		if len(roomPeersManager.peers) == 0 {
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
	peers         map[string]peer
	jitterHandler JitterHandler
}

func NewRoomPeersManager(loggerFactory LoggerFactory, jitterHandler JitterHandler) *RoomPeersManager {
	return &RoomPeersManager{
		loggerFactory: loggerFactory,
		log:           loggerFactory.GetLogger("roompeersmanager"),
		peers:         map[string]peer{},
		jitterHandler: jitterHandler,
	}
}

type peer struct {
	trackListener   *trackListener
	dataTransceiver *DataTransceiver
	signaller       *Signaller
}

func (t *RoomPeersManager) addTrack(clientID string, track *webrtc.Track) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for otherClientID, otherPeerInRoom := range t.peers {
		if otherClientID != clientID {
			if err := t.addTrackToPeer(otherPeerInRoom, clientID, track); err != nil {
				t.log.Printf("[%s] MemoryTracksManager.addTrack Error adding track: %s", otherClientID, err)
				continue
			}
		}
	}
}

func (t *RoomPeersManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for otherClientID, otherPeerInRoom := range t.peers {
		if otherClientID != clientID {
			t.log.Printf("[%s] broadcast from %s", otherClientID, clientID)
			tr := otherPeerInRoom.dataTransceiver
			var err error
			if msg.IsString {
				err = tr.SendText(string(msg.Data))
			} else {
				err = tr.Send(msg.Data)
			}
			if err != nil {
				t.log.Printf("[%s] broadcast error: %s", otherClientID, err)
			}
		}
	}
}

func (t *RoomPeersManager) addTrackToPeer(p peer, sourceClientID string, track *webrtc.Track) error {
	trackListener := p.trackListener
	rtcpCh, err := trackListener.AddTrack(sourceClientID, track)
	if err != nil {
		return fmt.Errorf("[%s] addTrackToPeer Error adding track: %d: %s", trackListener.ClientID(), track.SSRC(), err)
	}

	go func() {
		for pkt := range rtcpCh {
			t.mu.Lock()
			sourcePeer, ok := t.peers[sourceClientID]
			if !ok {
				t.log.Printf("[%s] addTrackToPeer error sending RTCP packet to source peer: %s. Peer not found", p.trackListener.clientID, sourceClientID)
				// do not return early since the rtcp channel needs to be emptied
			} else {
				err := sourcePeer.trackListener.WriteRTCP(pkt)
				if err != nil {
					t.log.Printf("[%s] addTrackToPeer error sending RTCP packet to source peer: %s. %s", p.trackListener.clientID, sourceClientID, err)
					// do not return early since the rtcp channel needs to be emptied
				}
			}
			t.mu.Unlock()
		}
	}()

	kind := track.Kind()
	signaller := p.signaller
	if signaller.Initiator() {
		t.log.Printf("[%s] addTrackToPeer Calling signaller.Negotiate() because a new %s track was added", trackListener.ClientID(), kind)
		signaller.Negotiate()
	} else {
		t.log.Printf("[%s] addTrackToPeer Calling signaller.AddTransceiverRequest() because a new %s track was added", trackListener.ClientID(), kind)
		signaller.SendTransceiverRequest(kind, webrtc.RTPTransceiverDirectionRecvonly)
	}
	return nil
}

func (t *RoomPeersManager) Add(
	clientID string,
	peerConnection *webrtc.PeerConnection,
	dataChannel *webrtc.DataChannel,
	signaller *Signaller,
) {
	onTrackEvent := func(e TrackEvent) {
		switch e.Type {
		case TrackEventTypeAdd:
			t.addTrack(e.ClientID, e.Track)
		case TrackEventTypeRemove:
			t.removeTrack(e.ClientID, e.Track)
		}
	}

	trackListener := newTrackListener(
		t.loggerFactory,
		clientID,
		peerConnection,
		onTrackEvent,
		t.jitterHandler,
	)

	t.mu.Lock()
	defer t.mu.Unlock()

	dataTransceiver := newDataTransceiver(t.loggerFactory, clientID, dataChannel, peerConnection)
	peerJoiningRoom := peer{trackListener, dataTransceiver, signaller}

	for existingPeerClientID, existingPeerInRoom := range t.peers {
		for _, track := range existingPeerInRoom.trackListener.Tracks() {
			// TODO what if tracks list changes in the meantime?
			err := t.addTrackToPeer(peerJoiningRoom, existingPeerClientID, track)
			if err != nil {
				t.log.Printf(
					"Error adding peer clientID: %s track to clientID: %s - reason: %s",
					existingPeerClientID,
					clientID,
					err,
				)
			}
		}
	}

	t.peers[clientID] = peerJoiningRoom

	messagesChannel := dataTransceiver.MessagesChannel()
	go func() {
		for msg := range messagesChannel {
			t.broadcast(clientID, msg)
		}
	}()
}

// GetTracksMetadata retrieves track metadata for a specific peer
func (t *RoomPeersManager) GetTracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	peer, ok := t.peers[clientID]
	if !ok {
		return m, false
	}
	m = peer.trackListener.GetTracksMetadata()
	return m, true
}

func (t *RoomPeersManager) Remove(clientID string) {
	t.log.Printf("removePeer: %s", clientID)
	t.mu.Lock()
	defer t.mu.Unlock()
	peerLeavingRoom, ok := t.peers[clientID]
	if !ok {
		t.log.Printf("Cannot remove peer clientID: %s (not found)", clientID)
		return
	}

	peerLeavingRoom.dataTransceiver.Close()

	delete(t.peers, clientID)
}

func (t *RoomPeersManager) removeTrack(clientID string, track *webrtc.Track) {
	t.log.Printf("[%s] removeTrack ssrc: %d from other peers", clientID, track.SSRC())

	t.mu.Lock()
	defer t.mu.Unlock()

	for otherClientID, otherPeerInRoom := range t.peers {
		if otherClientID != clientID {
			err := otherPeerInRoom.trackListener.RemoveTrack(track)
			if err != nil {
				t.log.Printf("[%s] removeTrack error removing track: %s", clientID, err)
			}
			otherPeerInRoom.signaller.Negotiate()
		}
	}
}
