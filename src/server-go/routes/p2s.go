package routes

import (
	"net/http"

	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
)

func NewPeer2ServerRoomHandler(wss *wsserver.WSS) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		wss.HandleRoom(w, r, func(event wsserver.RoomEvent) {
		})
	}
	return http.HandlerFunc(fn)
}
