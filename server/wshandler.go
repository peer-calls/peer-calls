package server

import (
	"context"
	"errors"
	"net/http"
	"path"
	"time"

	"nhooyr.io/websocket"
)

type WSS struct {
	log   Logger
	rooms RoomManager
}

func NewWSS(
	loggerFactory LoggerFactory,
	rooms RoomManager,
) *WSS {
	return &WSS{
		log:   loggerFactory.GetLogger("wss"),
		rooms: rooms,
	}
}

type RoomEvent struct {
	ClientID string
	Room     string
	Adapter  Adapter
	Message  Message
}

type CleanupEvent struct {
	ClientID string
	Room     string
	Adapter  Adapter
}

func (wss *WSS) HandleRoom(w http.ResponseWriter, r *http.Request, handleMessage func(RoomEvent)) {
	wss.HandleRoomWithCleanup(w, r, handleMessage, nil)
}

func (wss *WSS) HandleRoomWithCleanup(w http.ResponseWriter, r *http.Request, handleMessage func(RoomEvent), cleanup func(CleanupEvent)) {
	var err error

	start := time.Now()
	prometheusWSConnTotal.Inc()
	prometheusWSConnActive.Inc()
	defer func() {
		prometheusWSConnActive.Dec()
		if err != nil {
			prometheusWSConnErrTotal.Inc()
		}
		duration := time.Now().Sub(start)
		prometheusWSConnDuration.Observe(duration.Seconds())
	}()

	var c *websocket.Conn
	c, err = websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		wss.log.Printf("Error accepting websocket connection: %s", err)
		return
	}

	clientID := path.Base(r.URL.Path)
	room := path.Base(path.Dir(r.URL.Path))

	defer func() {
		wss.log.Printf("Closing websocket connection room: %s, clientID: %s", room, clientID)
		c.Close(websocket.StatusInternalError, "")
	}()
	ctx := r.Context()

	client := NewClientWithID(c, clientID)
	wss.log.Printf("New websocket connection - room: %s, clientID: %s", room, clientID)

	adapter := wss.rooms.Enter(room)
	defer func() {
		wss.log.Printf("wss.rooms.Exit room: %s, clientID: %s", room, clientID)
		wss.rooms.Exit(room)
	}()
	err = adapter.Add(client)
	if err != nil {
		wss.log.Printf("Error adding client to room: %s", err)
		return
	}

	if cleanup != nil {
		defer cleanup(CleanupEvent{
			ClientID: clientID,
			Room:     room,
			Adapter:  adapter,
		})
	}

	defer func() {
		wss.log.Printf("adapter.Remove room: %s, clientID: %s", room, clientID)
		err := adapter.Remove(clientID)
		if err != nil {
			wss.log.Printf("Error removing client from adapter: %s", err)
		}
	}()

	msgChan := client.Subscribe(ctx)

	for message := range msgChan {
		handleMessage(RoomEvent{
			ClientID: clientID,
			Room:     room,
			Adapter:  adapter,
			Message:  message,
		})
	}
	err = client.Err()

	if errors.Is(err, context.Canceled) {
		err = nil
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		err = nil
		return
	}
	if err != nil {
		wss.log.Printf("Subscription error: %s", err)
	}
}
