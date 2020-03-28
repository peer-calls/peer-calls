package wsserver

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

type RoomEvent struct {
	ClientID string
	Room     string
	Adapter  wsadapter.Adapter
	Message  wsmessage.Message
}

func (wss *WSS) HandleRoom(w http.ResponseWriter, r *http.Request, handleMessage func(RoomEvent)) {
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

	err = client.Subscribe(ctx, func(message wsmessage.Message) {
		handleMessage(RoomEvent{
			ClientID: clientID,
			Room:     room,
			Adapter:  adapter,
			Message:  message,
		})
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
