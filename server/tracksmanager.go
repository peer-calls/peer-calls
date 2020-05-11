package server

import (
	"fmt"
	"sync"

	"github.com/pion/webrtc/v2"
)

const DataChannelName = "data"

type MemoryTracksManager struct {
	loggerFactory LoggerFactory
	log           Logger
	mu            sync.RWMutex
	// key is clientID
	peers map[string]peer
	// key is room, value is clientID
	peerIDsByRoom map[string]map[string]struct{}
}

func NewMemoryTracksManager(loggerFactory LoggerFactory) *MemoryTracksManager {
	return &MemoryTracksManager{
		loggerFactory: loggerFactory,
		log:           loggerFactory.GetLogger("tracksmanager"),
		peers:         map[string]peer{},
		peerIDsByRoom: map[string]map[string]struct{}{},
	}
}

type peer struct {
	trackListener   *trackListener
	dataTransceiver *DataTransceiver
	room            string
	signaller       *Signaller
}

func (t *MemoryTracksManager) addTrack(room string, clientID string, track *webrtc.Track) {
	t.mu.Lock()

	peersSet := t.peerIDsByRoom[room]
	for otherClientID := range peersSet {
		otherPeerInRoom := t.peers[otherClientID]
		if otherClientID != clientID {
			if err := t.addTrackToPeer(otherPeerInRoom, clientID, track); err != nil {
				t.log.Printf("[%s] MemoryTracksManager.addTrack Error adding track: %s", otherClientID, err)
				continue
			}
		}
	}

	t.mu.Unlock()
}

func (t *MemoryTracksManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
	t.mu.Lock()

	peer, ok := t.peers[clientID]
	if !ok {
		t.log.Printf("[%s] broadcast peer not found", clientID)
		return
	}

	peersSet := t.peerIDsByRoom[peer.room]
	for otherClientID := range peersSet {
		otherPeerInRoom := t.peers[otherClientID]
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

	t.mu.Unlock()
}

func (t *MemoryTracksManager) addTrackToPeer(p peer, sourceClientID string, track *webrtc.Track) error {
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

func (t *MemoryTracksManager) Add(
	room string,
	clientID string,
	peerConnection *webrtc.PeerConnection,
	dataChannel *webrtc.DataChannel,
	signaller *Signaller,
) {
	t.log.Printf("[%s] TrackManager.Add peer to room: %s", clientID, room)

	onTrackEvent := func(e TrackEvent) {
		switch e.Type {
		case TrackEventTypeAdd:
			t.addTrack(room, e.ClientID, e.Track)
		case TrackEventTypeRemove:
			t.removeTrack(room, e.ClientID, e.Track)
		}
	}

	trackListener := newTrackListener(
		t.loggerFactory,
		clientID,
		peerConnection,
		onTrackEvent,
	)

	t.mu.Lock()
	dataTransceiver := newDataTransceiver(t.loggerFactory, clientID, dataChannel, peerConnection)
	peerJoiningRoom := peer{trackListener, dataTransceiver, room, signaller}

	peersSet, ok := t.peerIDsByRoom[room]
	if !ok {
		peersSet = map[string]struct{}{}
		t.peerIDsByRoom[room] = peersSet
	}

	for existingPeerClientID := range peersSet {
		existingPeerInRoom, ok := t.peers[existingPeerClientID]
		if !ok {
			t.log.Printf("[%s] Cannot find existing peer", existingPeerClientID)
			continue
		}
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
	peersSet[clientID] = struct{}{}

	messagesChannel := dataTransceiver.MessagesChannel()
	go func() {
		for msg := range messagesChannel {
			t.broadcast(clientID, msg)
		}
	}()

	go func() {
		<-signaller.CloseChannel()
		t.removePeer(clientID)
	}()

	t.mu.Unlock()
}

// GetTracksMetadata retrieves track metadata for a specific peer
func (t *MemoryTracksManager) GetTracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	peer, ok := t.peers[clientID]
	if !ok {
		return m, false
	}
	m = peer.trackListener.GetTracksMetadata()
	return m, true
}

func (t *MemoryTracksManager) removePeer(clientID string) {
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
	peerIDs, ok := t.peerIDsByRoom[peerLeavingRoom.room]
	if !ok {
		t.log.Printf("Cannot remove peer ID from room: %s (not found)", clientID)
		return
	}
	delete(peerIDs, clientID)
	if len(peerIDs) == 0 {
		delete(t.peerIDsByRoom, peerLeavingRoom.room)
	}
}

func (t *MemoryTracksManager) removeTrack(room string, clientID string, track *webrtc.Track) {
	t.log.Printf("[%s] removeTrack ssrc: %d from other peers", clientID, track.SSRC())

	t.mu.Lock()
	defer t.mu.Unlock()

	clientIDs, ok := t.peerIDsByRoom[room]
	if !ok {
		t.log.Printf("[%s] removeTrack: Cannot find any peers in room: %s", clientID, room)
		return
	}
	for otherClientID := range clientIDs {
		if otherClientID != clientID {
			otherPeerInRoom := t.peers[otherClientID]
			err := otherPeerInRoom.trackListener.RemoveTrack(track)
			if err != nil {
				t.log.Printf("[%s] removeTrack error removing track: %s", clientID, err)
			}
			otherPeerInRoom.signaller.Negotiate()
		}
	}
}
