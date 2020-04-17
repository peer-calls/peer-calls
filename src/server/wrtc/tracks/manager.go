package tracks

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jeremija/peer-calls/src/server/logger"
	"github.com/pion/webrtc/v2"
)

const DataChannelName = "data"

var log = logger.GetLogger("tracks")

type TracksManager struct {
	mu sync.RWMutex
	// key is clientID
	peers map[string]peerInRoom
	// key is room, value is clientID
	peerIDsByRoom map[string]map[string]struct{}
}

type Signaller interface {
	Initiator() bool
	SendTransceiverRequest(kind webrtc.RTPCodecType, direction webrtc.RTPTransceiverDirection)
	Negotiate()
	CloseChannel() <-chan struct{}
}

func NewTracksManager() *TracksManager {
	return &TracksManager{
		peers:         map[string]peerInRoom{},
		peerIDsByRoom: map[string]map[string]struct{}{},
	}
}

type peerInRoom struct {
	peer            *peer
	dataTransceiver *DataTransceiver
	room            string
	signaller       Signaller
}

func (t *TracksManager) addTrack(room string, clientID string, track *webrtc.Track) {
	t.mu.Lock()

	for otherClientID, otherPeerInRoom := range t.peers {
		if otherClientID != clientID {
			if err := addTrackToPeer(otherPeerInRoom, track); err != nil {
				log.Printf("[%s] TracksManager.addTrack Error adding track: %s", otherClientID, err)
				continue
			}
		}
	}

	t.mu.Unlock()
}

func (t *TracksManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
	t.mu.Lock()

	for otherClientID, otherPeerInRoom := range t.peers {
		if otherClientID != clientID {
			log.Printf("[%s] broadcast from %s", otherClientID, clientID)
			tr := otherPeerInRoom.dataTransceiver
			var err error
			if msg.IsString {
				textData := msg.Data
				data := map[string]interface{}{}
				if unmarshalErr := json.Unmarshal(textData, &data); unmarshalErr == nil {
					data["userId"] = clientID
					textData, _ = json.Marshal(data)
				}
				err = tr.SendText(string(textData))
			} else {
				err = tr.Send(msg.Data)
			}
			if err != nil {
				log.Printf("[%s] broadcast error: %s", otherClientID, err)
			}
		}
	}

	t.mu.Unlock()
}

func addTrackToPeer(peerInRoom peerInRoom, track *webrtc.Track) error {
	peer := peerInRoom.peer
	if err := peer.AddTrack(track); err != nil {
		return fmt.Errorf("[%s] addTrackToPeer Error adding track: %s: %s", peer.ClientID(), track.ID(), err)
	}

	kind := track.Kind()
	signaller := peerInRoom.signaller
	log.Printf("[%s] addTrackToPeer Calling signaller.AddTransceiverRequest() and signaller.Negotiate() because a new %s track was added", peer.ClientID(), kind)
	if signaller.Initiator() {
		signaller.Negotiate()
	} else {
		signaller.SendTransceiverRequest(kind, webrtc.RTPTransceiverDirectionRecvonly)
	}
	return nil
}

func (t *TracksManager) Add(
	room string,
	clientID string,
	peerConnection PeerConnection,
	dataChannel *webrtc.DataChannel,
	signaller Signaller,
) (closeChannel <-chan struct{}) {
	log.Printf("[%s] TrackManager.Add peer to room: %s", clientID, room)

	peer := newPeer(
		clientID,
		peerConnection,
	)

	t.mu.Lock()
	dataTransceiver := newDataTransceiver(clientID, dataChannel, peerConnection)
	peerJoiningRoom := peerInRoom{peer, dataTransceiver, room, signaller}

	peersSet, ok := t.peerIDsByRoom[room]
	if !ok {
		peersSet = map[string]struct{}{}
		t.peerIDsByRoom[room] = peersSet
	}

	for existingPeerClientID := range peersSet {
		existingPeerInRoom, ok := t.peers[existingPeerClientID]
		if !ok {
			log.Printf("[%s] Cannot find existing peer", existingPeerClientID)
			continue
		}
		for _, track := range existingPeerInRoom.peer.Tracks() {
			// TODO what if tracks list changes in the meantime?
			err := addTrackToPeer(peerJoiningRoom, track)
			if err != nil {
				log.Printf(
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

	tracksChannel := peer.TracksChannel()
	go func() {
		for e := range tracksChannel {
			switch e.Type {
			case TrackEventTypeAdd:
				t.addTrack(room, e.ClientID, e.Track)
			case TrackEventTypeRemove:
				t.removeTrack(e.ClientID, e.Track)
			}
		}
	}()

	go func() {
		<-signaller.CloseChannel()
		t.removePeer(clientID)
	}()

	t.mu.Unlock()

	return signaller.CloseChannel()
}

func (t *TracksManager) removePeer(clientID string) {
	log.Printf("removePeer: %s", clientID)
	t.mu.Lock()
	defer t.mu.Unlock()
	peerLeavingRoom, ok := t.peers[clientID]
	if !ok {
		log.Printf("Cannot remove peer clientID: %s (not found)", clientID)
		return
	}

	peerLeavingRoom.peer.Close()
	peerLeavingRoom.dataTransceiver.Close()
	t.removePeerTracks(peerLeavingRoom)

	delete(t.peers, clientID)
	peerIDs, ok := t.peerIDsByRoom[peerLeavingRoom.room]
	if !ok {
		log.Printf("Cannot remove peer ID from room: %s (not found)", clientID)
		return
	}
	delete(peerIDs, clientID)
}

func (t *TracksManager) removePeerTracks(peerLeavingRoom peerInRoom) {
	leavingClientID := peerLeavingRoom.peer.ClientID()
	log.Printf("Remove all peer tracks for clientID: %s", leavingClientID)
	clientIDs, ok := t.peerIDsByRoom[peerLeavingRoom.room]
	if !ok {
		log.Println("Cannot find any peers in room", peerLeavingRoom.room)
		return
	}

	tracks := peerLeavingRoom.peer.Tracks()
	for clientID := range clientIDs {
		if clientID != leavingClientID {
			otherPeerInRoom := t.peers[clientID]
			for _, track := range tracks {
				log.Printf(
					"Removing track: %s from peer clientID: %s (source clientID: %s)",
					track.ID(),
					clientID,
					leavingClientID,
				)
				err := otherPeerInRoom.peer.RemoveTrack(track)
				if err != nil {
					log.Printf(
						"Error removing track: %s from peer clientID: %s (source clientID: %s): %s",
						track.ID(),
						clientID,
						leavingClientID,
						err,
					)
				}
			}
			otherPeerInRoom.signaller.Negotiate()
		}
	}
}

func (t *TracksManager) removeTrack(clientID string, track *webrtc.Track) {
	log.Printf("[%s] removeTrack ssrc: %d from other peers", clientID, track.SSRC())

	t.mu.Lock()
	defer t.mu.Unlock()

	peer, ok := t.peers[clientID]
	if !ok {
		log.Printf("[%s] removeTrack: Cannot find peer with clientID: %s", clientID)
		return
	}
	clientIDs, ok := t.peerIDsByRoom[peer.room]
	if !ok {
		log.Printf("[%s] removeTrack: Cannot find any peers in room: %s", clientID, peer.room)
		return
	}
	for otherClientID := range clientIDs {
		if otherClientID != clientID {
			otherPeerInRoom := t.peers[otherClientID]
			err := otherPeerInRoom.peer.RemoveTrack(track)
			if err != nil {
				log.Printf("[%s] removeTrack error removing track: %s", clientID, err)
			}
			otherPeerInRoom.signaller.Negotiate()
		}
	}
}
