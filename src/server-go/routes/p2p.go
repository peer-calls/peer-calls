package routes

import (
	"net/http"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
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

func NewPeerToPeerRoomHandler(wss *wsserver.WSS) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		wss.HandleRoom(w, r, func(event wsserver.RoomEvent) {
			msg := event.Message
			adapter := event.Adapter
			room := event.Room
			clientID := event.ClientID

			var responseEventName string
			var err error

			switch msg.Type {
			case "ready":
				clients, err := adapter.Clients()
				if err != nil {
					log.Printf("Error retrieving clients: %s", err)
				}
				responseEventName = "users"
				log.Printf("Got clients: %s", clients)
				err = adapter.Broadcast(
					wsmessage.NewMessage(responseEventName, room, map[string]interface{}{
						"initiator": clientID,
						"users":     clientsToUsers(clients),
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

type User struct {
	UserID   string `json:"userId"`
	ClientID string `json:"clientId"`
}

func clientsToUsers(clients map[string]string) (users []User) {
	for clientID := range clients {
		users = append(users, User{
			UserID:   clientID,
			ClientID: clientID,
		})
	}
	return
}
