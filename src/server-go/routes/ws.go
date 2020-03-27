package routes

import (
	"context"
	"errors"
	"net/http"
	"path"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/ws"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsadapter"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"nhooyr.io/websocket"
)

type AdapterFactory func(room string) wsadapter.Adapter

var log = logger.GetLogger("ws")

type RoomManager interface {
	Enter(room string) wsadapter.Adapter
	Exit(room string)
}

type WSS struct {
	rooms RoomManager
}

func NewWSS(rooms RoomManager) *WSS {
	return &WSS{
		rooms: rooms,
	}
}

type ReadyMessage struct {
	UserID string `json:"userId"`
	Room   string `json:"room"`
}

func (wss *WSS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("Error accepting websocket connection: %s", err)
		return
	}

	clientID := path.Base(r.URL.Path)
	room := path.Base(path.Dir(r.URL.Path))

	defer func() {
		log.Printf("Closing websocket connection room: %s, clientID: %s", room, clientID)
		c.Close(websocket.StatusInternalError, "")
	}()
	ctx := r.Context()

	client := ws.NewClientWithID(c, clientID)
	defer client.Close()
	log.Printf("New websocket connection - room: %s, clientID: %s", room, clientID)

	adapter := wss.rooms.Enter(room)
	defer func() {
		log.Printf("wss.rooms.Exit room: %s, clientID: %s", room, clientID)
		wss.rooms.Exit(room)
	}()
	err = adapter.Add(client)
	if err != nil {
		log.Printf("Error adding client to room: %s", err)
		return
	}

	defer func() {
		log.Printf("adapter.Remove room: %s, clientID: %s", room, clientID)
		err := adapter.Remove(clientID)
		if err != nil {
			log.Printf("Error removing client from adapter: %s", err)
		}
	}()

	err = client.Subscribe(ctx, func(msg wsmessage.Message) {
		switch msg.Type {
		case "ready":
			clients, err := adapter.Clients()
			if err != nil {
				log.Printf("Error retrieving clients: %s", err)
			}
			log.Printf("Got clients: %s", clients)
			adapter.Broadcast(
				wsmessage.NewMessage("users", room, map[string]interface{}{
					"initiator": clientID,
					"users":     clientsToUsers(clients),
				}),
			)
		case "signal":
			payload, _ := msg.Payload.(map[string]interface{})
			signal, _ := payload["signal"].(string)
			targetClientID, _ := payload["userId"].(string)

			log.Printf("Send signal from: %s to %s", clientID, targetClientID)
			adapter.Emit(targetClientID, wsmessage.NewMessage("signal", room, map[string]string{
				"userId": clientID,
				"signal": signal,
			}))
		}
	})

	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		log.Printf("Subscription error: %s", err)
	}
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
