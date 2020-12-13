package server

import (
	"context"
	pkgErrors "errors"
	"net/http"
	"path"
	"time"

	"github.com/juju/errors"
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

type Subscription struct {
	Adapter  Adapter
	ClientID string
	Room     string
	Messages <-chan Message
}

func (wss *WSS) Subscribe(w http.ResponseWriter, r *http.Request) (*Subscription, error) {
	var c *websocket.Conn
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})

	if err != nil {
		prometheusWSConnErrTotal.Inc()
		return nil, errors.Annotatef(err, "accept websocket connection")
	}

	ctx := r.Context()

	clientID := path.Base(r.URL.Path)
	room := path.Base(path.Dir(r.URL.Path))
	adapter := wss.rooms.Enter(room)
	ch := make(chan Message)

	client := NewClientWithID(c, clientID)
	wss.log.Printf("[%s] New websocket connection - room: %s", clientID, room)

	prometheusWSConnTotal.Inc()
	prometheusWSConnActive.Inc()
	start := time.Now()

	go func() {
		defer func() {
			prometheusWSConnActive.Dec()
			duration := time.Now().Sub(start)
			prometheusWSConnDuration.Observe(duration.Seconds())

			wss.log.Printf("[%s] Closing websocket connection - room: %s", clientID, room)
			err := c.Close(websocket.StatusNormalClosure, "")
			if err != nil {
				err = errors.Trace(err)
				wss.log.Printf("[%s] Error closing websocket connection: %+v", clientID, err)
			}
		}()
		defer func() {
			wss.log.Printf("[%s] wss.rooms.Exit room: %s", clientID, room)
			wss.rooms.Exit(room)
		}()
		err = adapter.Add(client)
		if err != nil {
			err = errors.Trace(err)
			wss.log.Printf("[%s] Error adding client to room: %s: %+v", clientID, room, err)
			close(ch)
			return
		}

		defer func() {
			wss.log.Printf("[%s] adapter.Remove room: %s", clientID, room)
			err := adapter.Remove(clientID)
			if err != nil {
				wss.log.Printf("[%s] Error removing client from adapter: %+v", clientID, err)
			}
		}()

		msgChan := client.Subscribe(ctx)

		for message := range msgChan {
			ch <- message
		}

		close(ch)

		err = errors.Trace(client.Err())
		cause := errors.Cause(err)

		if pkgErrors.Is(cause, context.Canceled) {
			err = nil
			return
		}
		if websocket.CloseStatus(cause) == websocket.StatusNormalClosure ||
			websocket.CloseStatus(cause) == websocket.StatusGoingAway {
			err = nil
			return
		}

		if err != nil {
			wss.log.Printf("[%s] Subscription error: %+v", clientID, err)
		}
	}()

	stream := &Subscription{
		Adapter:  adapter,
		ClientID: clientID,
		Room:     room,
		Messages: ch,
	}

	return stream, nil
}
