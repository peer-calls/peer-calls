package server

import (
	"context"
	pkgErrors "errors"
	"net/http"
	"path"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/message"
	"nhooyr.io/websocket"
)

type WSS struct {
	log   logger.Logger
	rooms RoomManager
}

func NewWSS(log logger.Logger, rooms RoomManager) *WSS {
	return &WSS{
		log:   log.WithNamespaceAppended("wss"),
		rooms: rooms,
	}
}

type Subscription struct {
	Adapter  Adapter
	ClientID identifiers.ClientID
	Room     identifiers.RoomID
	Messages <-chan message.Message
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

	clientID := identifiers.ClientID(path.Base(r.URL.Path))
	room := identifiers.RoomID(path.Base(path.Dir(r.URL.Path)))
	adapter, _ := wss.rooms.Enter(room)
	ch := make(chan message.Message)

	log := wss.log.WithCtx(logger.Ctx{
		"client_id": clientID,
		"room_id":   room,
	})

	client := NewClientWithID(c, clientID)

	log.Info("New websocket connection", nil)

	prometheusWSConnTotal.Inc()
	prometheusWSConnActive.Inc()
	start := time.Now()

	go func() {
		defer func() {
			prometheusWSConnActive.Dec()
			duration := time.Now().Sub(start)
			prometheusWSConnDuration.Observe(duration.Seconds())

			err := c.Close(websocket.StatusNormalClosure, "")
			if err != nil {
				log.Error("Close websocket connection", errors.Trace(err), nil)
			} else {
				log.Info("Close websocket connection", nil)
			}
		}()
		defer func() {
			log.Info("Exit", nil)
			wss.rooms.Exit(room)
		}()
		err = adapter.Add(client)
		if err != nil {
			log.Error("Add client", errors.Trace(err), nil)

			close(ch)

			return
		}

		defer func() {
			err := adapter.Remove(clientID)
			if err != nil {
				log.Error("Remove", errors.Trace(err), nil)
			} else {
				log.Info("Remove", nil)
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
			log.Error("Subscribe", errors.Trace(err), nil)
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
