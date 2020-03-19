package routes_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/routes"
	"github.com/stretchr/testify/require"
)

func TestRouteCall_name(t *testing.T) {
	mux := routes.NewMux("/test", "v0.0.0", "[]")
	w := httptest.NewRecorder()
	reader := strings.NewReader("call=my room")
	r := httptest.NewRequest("POST", "/test/call", reader)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(w, r)

	require.Equal(t, 302, w.Code, "expected 302 redirect")
	require.Equal(t, "/test/call/my%20room", w.Header().Get("Location"))
}

func TestRouteCall_random(t *testing.T) {
	mux := routes.NewMux("/test", "v0.0.0", "[]")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test/call", nil)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(w, r)

	uuid := "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
	require.Equal(t, 302, w.Code, "expected 302 redirect")
	require.Regexp(t, "/test/call/"+uuid, w.Header().Get("Location"))
}
