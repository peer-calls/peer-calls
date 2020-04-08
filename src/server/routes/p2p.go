package routes

import (
	"net/http"

	"github.com/jeremija/peer-calls/src/server/logger"
	"github.com/jeremija/peer-calls/src/server/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server/ws/wsmessage"
	"github.com/jeremija/peer-calls/src/server/wshandler"
)

type AdapterFactory func(room string) wsadapter.Adapter

var log = logger.GetLogger("ws")

type RoomManager interface {
	Enter(room string) wsadapter.Adapter
	Exit(room string)
}

type ReadyMessage struct {
	UserID string `json:"userId"`
	Room   string `json:"room"`
}

func NewPeerToPeerRoomHandler(wss *wshandler.WSS) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		wss.HandleRoom(w, r, func(event wshandler.RoomEvent) {
			msg := event.Message
			adapter := event.Adapter
			room := event.Room
			clientID := event.ClientID

			var responseEventName string
			var err error

			switch msg.Type {
			case "ready":
				// FIXME check for errors
				payload, _ := msg.Payload.(map[string]interface{})
				adapter.SetMetadata(clientID, payload["nickname"].(string))

				clients, err := adapter.Clients()
				if err != nil {
					log.Printf("Error retrieving clients: %s", err)
				}
				responseEventName = "users"
				log.Printf("Got clients: %s", clients)
				err = adapter.Broadcast(
					wsmessage.NewMessage(responseEventName, room, map[string]interface{}{
						"initiator": clientID,
						"peerIds":   clientsToPeerIDs(clients),
						"nicknames": clients,
					}),
				)
			case "signal":
				payload, _ := msg.Payload.(map[string]interface{})
				signal, _ := payload["signal"]
				targetClientID, _ := payload["userId"].(string)

				responseEventName = "signal"
				log.Printf("Send signal from: %s to %s", clientID, targetClientID)
				err = adapter.Emit(targetClientID, wsmessage.NewMessage(responseEventName, room, map[string]interface{}{
					"userId": clientID,
					"signal": signal,
				}))
			}

			if err != nil {
				log.Printf("Error sending event (event: %s, room: %s, source: %s)", responseEventName, room, clientID)
			}
		})
	}
	return http.HandlerFunc(fn)
}

func clientsToPeerIDs(clients map[string]string) (peers []string) {
	for clientID := range clients {
		peers = append(peers, clientID)
	}
	return
}
