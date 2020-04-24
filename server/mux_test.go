package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/pion/webrtc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var iceServers = []server.ICEServer{}

type addedPeer struct {
	room           string
	clientID       string
	peerConnection *webrtc.PeerConnection
}

type mockTracksManager struct {
	added chan addedPeer
}

func newMockTracksManager() *mockTracksManager {
	return &mockTracksManager{
		added: make(chan addedPeer, 10),
	}
}

func (m *mockTracksManager) Add(room string, clientID string, peerConnection *webrtc.PeerConnection, dataChannel *webrtc.DataChannel, signaller *server.Signaller) {
	m.added <- addedPeer{
		room:           room,
		clientID:       clientID,
		peerConnection: peerConnection,
	}
}

func (m *mockTracksManager) GetTracksMetadata(clientID string) ([]server.TrackMetadata, bool) {
	return nil, true
}

func mesh() (network server.NetworkConfig) {
	network.Type = server.NetworkTypeMesh
	return
}

func Test_routeIndex(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	mux.ServeHTTP(w, r)

	require.Equal(t, 200, w.Code)
	require.Regexp(t, "action=\"/test/call\"", w.Body.String())
}

func Test_routeIndex_noBaseURL(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "", "v0.0.0", mesh(), iceServers, mrm, trk)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mux.ServeHTTP(w, r)

	require.Equal(t, 200, w.Code)
	require.Regexp(t, "action=\"/call\"", w.Body.String())
}

func Test_routeNewCall_name(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk)
	w := httptest.NewRecorder()
	reader := strings.NewReader("call=my room")
	r := httptest.NewRequest("POST", "/test/call", reader)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(w, r)

	require.Equal(t, 302, w.Code, "expected 302 redirect")
	require.Equal(t, "/test/call/my%20room", w.Header().Get("Location"))
}

func Test_routeNewCall_random(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test/call", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(w, r)

	uuid := "[0-9a-z-A-Z]+$"
	require.Equal(t, 302, w.Code, "expected 302 redirect")
	require.Regexp(t, "/test/call/"+uuid, w.Header().Get("Location"))
}

func Test_routeCall(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	iceServers := []server.ICEServer{{
		URLs: []string{"stun:"},
	}}
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test/call/abc", nil)
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Regexp(t, "id=\"baseUrl\" value=\"/test\"", w.Body.String())
	assert.Regexp(t, "id=\"callId\" value=\"abc\"", w.Body.String())
	assert.Regexp(t, "id=\"iceServers\" value='.*stun:", w.Body.String())
	assert.Regexp(t, "id=\"userId\" value=\"[^\"]", w.Body.String())
}
