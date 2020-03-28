package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/routes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var iceServers = []config.ICEServer{}

func Test_routeIndex(t *testing.T) {
	mrm := NewMockRoomManager()
	defer mrm.close()
	mux := routes.NewMux("/test", "v0.0.0", iceServers, "[]", mrm)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	mux.ServeHTTP(w, r)

	require.Equal(t, 200, w.Code)
	require.Regexp(t, "action=\"/test/call\"", w.Body.String())
}

func Test_routeIndex_noBaseURL(t *testing.T) {
	mrm := NewMockRoomManager()
	defer mrm.close()
	mux := routes.NewMux("", "v0.0.0", iceServers, "[]", mrm)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	mux.ServeHTTP(w, r)

	require.Equal(t, 200, w.Code)
	require.Regexp(t, "action=\"/call\"", w.Body.String())
}

func Test_routeNewCall_name(t *testing.T) {
	mrm := NewMockRoomManager()
	defer mrm.close()
	mux := routes.NewMux("/test", "v0.0.0", iceServers, "[]", mrm)
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
	defer mrm.close()
	mux := routes.NewMux("/test", "v0.0.0", iceServers, "[]", mrm)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test/call", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(w, r)

	uuid := "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
	require.Equal(t, 302, w.Code, "expected 302 redirect")
	require.Regexp(t, "/test/call/"+uuid, w.Header().Get("Location"))
}

func Test_routeCall(t *testing.T) {
	mrm := NewMockRoomManager()
	defer mrm.close()
	mux := routes.NewMux("/test", "v0.0.0", iceServers, "[{\"urls\":\"stun:\"}]", mrm)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test/call/abc", nil)
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Regexp(t, "id=\"baseUrl\" value=\"/test\"", w.Body.String())
	assert.Regexp(t, "id=\"callId\" value=\"abc\"", w.Body.String())
	assert.Regexp(t, "id=\"iceServers\" value='.*stun:", w.Body.String())
	assert.Regexp(t, "id=\"userId\" value=\"[^\"]", w.Body.String())
}
