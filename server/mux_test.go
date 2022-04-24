package server_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/pubsub"
	"github.com/peer-calls/peer-calls/v4/server/sfu"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var iceServers = []server.ICEServer{}

type addedPeer struct {
	room      identifiers.RoomID
	clientID  identifiers.ClientID
	transport transport.Transport
}

type mockTracksManager struct {
	added        chan addedPeer
	subscribed   chan sfu.SubParams
	unsubscribed chan sfu.SubParams
}

var _ server.TracksManager = &mockTracksManager{}

func newMockTracksManager() *mockTracksManager {
	return &mockTracksManager{
		added:        make(chan addedPeer, 10),
		subscribed:   make(chan sfu.SubParams, 10),
		unsubscribed: make(chan sfu.SubParams, 10),
	}
}

func (m *mockTracksManager) Add(room identifiers.RoomID, transport transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	ch := make(chan pubsub.PubTrackEvent)
	close(ch)

	m.added <- addedPeer{
		room:      room,
		clientID:  clientID,
		transport: transport,
	}

	return ch, nil
}

func (m *mockTracksManager) Sub(params sfu.SubParams) error {
	m.subscribed <- params
	return nil
}

func (m *mockTracksManager) Unsub(params sfu.SubParams) error {
	m.unsubscribed <- params
	return nil
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
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom, embed)
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
	mux := server.NewMux(test.NewLogger(), "", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)
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
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)
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
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)
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
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test/call/abc", nil)
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	re := regexp.MustCompile(`id="config".*value="(.*?)"`)
	result := re.FindStringSubmatch(w.Body.String())

	var config server.ClientConfig
	err := json.Unmarshal([]byte(html.UnescapeString(result[1])), &config)
	require.NoError(t, err)

	assert.Equal(t, "/test", config.BaseURL)
	assert.Equal(t, "", config.Nickname)
	assert.Equal(t, "abc", config.CallID)
	assert.NotEmpty(t, config.PeerID)
	assert.NotEmpty(t, config.PeerConfig.ICEServers)
	assert.False(t, config.PeerConfig.EncodedInsertableStreams)
	assert.Equal(t, server.NetworkTypeMesh, config.Network)
}

func Test_manifest(t *testing.T) {
	mrm := NewMockRoomManager()
	trk := newMockTracksManager()
	defer mrm.close()
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)
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
	mux := server.NewMux(test.NewLogger(), "/test", "v0.0.0", mesh(), iceServers, false, mrm, trk, prom(), embed)

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
