package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var iceServers = []server.ICEServer{}

type addedPeer struct {
	room      string
	clientID  string
	transport *server.WebRTCTransport
}

type mockTracksManager struct {
	added chan addedPeer
}

var _ server.TracksManager = &mockTracksManager{}

func newMockTracksManager() *mockTracksManager {
	return &mockTracksManager{
		added: make(chan addedPeer, 10),
	}
}

func (m *mockTracksManager) Add(room string, transport *server.WebRTCTransport) {
	m.added <- addedPeer{
		room:      room,
		clientID:  clientID,
		transport: transport,
	}
}

func (m *mockTracksManager) GetTracksMetadata(room string, clientID string) ([]server.TrackMetadata, bool) {
	return nil, true
}

func mesh() (network server.NetworkConfig) {
	network.Type = server.NetworkTypeMesh
	return
}

const prometheusAccessToken = "prom1234"

func prom() server.PrometheusConfig {
	return server.PrometheusConfig{prometheusAccessToken}
}

func Test_routeIndex(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	prom := server.PrometheusConfig{"test1234"}
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom)
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
	mux := server.NewMux(loggerFactory, "", "v0.0.0", mesh(), iceServers, mrm, trk, prom())
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
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom())
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
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom())
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
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom())
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test/call/abc", nil)
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Regexp(t, "id=\"baseUrl\" value=\"/test\"", w.Body.String())
	assert.Regexp(t, "id=\"callId\" value=\"abc\"", w.Body.String())
	assert.Regexp(t, "id=\"iceServers\" value='.*stun:", w.Body.String())
	assert.Regexp(t, "id=\"userId\" value=\"[^\"]", w.Body.String())
}

func Test_manifest(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom())
	w := httptest.NewRecorder()
	reader := strings.NewReader("call=my room")
	r := httptest.NewRequest("GET", "/test/manifest.json", reader)
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	data := map[string]interface{}{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	assert.NoError(t, err)
}

func Test_Metrics(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(loggerFactory, "/test", "v0.0.0", mesh(), iceServers, mrm, trk, prom())

	for _, testCase := range []struct {
		statusCode    int
		authorization string
		url           string
	}{
		{401, "", "/test/metrics"},
		{401, "Bearer ", "/test/metrics"},
		{401, "Bearer", "/test/metrics"},
		{401, "Bearer invalid-token", "/test/metrics"},
		{200, "Bearer " + prometheusAccessToken, "/test/metrics"},
		{200, "", "/test/metrics?access_token=" + prometheusAccessToken},
		{401, "", "/test/metrics?access_token=invalid_token"},
	} {
		t.Run("URL: "+testCase.url+", Authorization: "+testCase.authorization, func(t *testing.T) {
			w := httptest.NewRecorder()
			reader := strings.NewReader("call=my room")
			r := httptest.NewRequest("GET", testCase.url, reader)
			r.Header.Set("Authorization", testCase.authorization)
			mux.ServeHTTP(w, r)
			assert.Equal(t, testCase.statusCode, w.Code)
		})
	}
}
