package routes

import (
	"net/http"

	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/ws/wsadapter"
	"github.com/peer-calls/peer-calls/server/ws/wsmessage"
	"github.com/peer-calls/peer-calls/server/wshandler"
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

				clients, err := getReadyClients(adapter)
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

func getReadyClients(adapter wsadapter.Adapter) (map[string]string, error) {
	filteredClients := map[string]string{}
	clients, err := adapter.Clients()
	if err != nil {
		return filteredClients, err
	}
	for clientID, nickname := range clients {
		// if nickame hasn't been set, the peer hasn't emitted ready yet so we
		// don't connect to that peer.
		if nickname != "" {
			filteredClients[clientID] = nickname
		}
	}
	return filteredClients, nil
}

func clientsToPeerIDs(clients map[string]string) (peers []string) {
	for clientID, _ := range clients {
		peers = append(peers, clientID)
	}
	return
}
