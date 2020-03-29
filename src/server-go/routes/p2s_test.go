package routes_test

import "github.com/jeremija/peer-calls/src/server-go/wrtc/tracks"

type addedPeer struct {
	room           string
	clientID       string
	peerConnection tracks.PeerConnection
}

type mockTracksManager struct {
	added chan addedPeer
}

func newMockTracksManager() *mockTracksManager {
	return &mockTracksManager{
		added: make(chan addedPeer, 10),
	}
}

func (m *mockTracksManager) Add(room string, clientID string, peerConnection tracks.PeerConnection) {
	m.added <- addedPeer{
		room:           room,
		clientID:       clientID,
		peerConnection: peerConnection,
	}
}
