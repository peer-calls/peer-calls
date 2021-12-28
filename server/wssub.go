package server

import (
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
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

type WebsocketContext struct {
	adapter   Adapter
	roomID    identifiers.RoomID
	client    *Client
	onClose   func()
	closeOnce sync.Once
}

// NewWebsocketContext initializes the new websocket context. Users must call
// the Close method once they are done.
func NewWebsocketContext(
	adapter Adapter, client *Client, roomID identifiers.RoomID, onClose func(),
) *WebsocketContext {
	return &WebsocketContext{
		adapter: adapter,
		roomID:  roomID,
		client:  client,
		onClose: onClose,
	}
}

// Adapter returns the websocket adapter.
func (w *WebsocketContext) Adapter() Adapter {
	return w.adapter
}

// RoomID returns the room identifier.
func (w *WebsocketContext) RoomID() identifiers.RoomID {
	return w.roomID
}

// ClientID return sthe client identifier.
func (w *WebsocketContext) ClientID() identifiers.ClientID {
	return w.client.ID()
}

// Messages returns the parsed messages channel.
func (w *WebsocketContext) Messages() <-chan message.Message {
	return w.client.Messages()
}

// Close invokes the Close method on the underlying connection. It also invokes
// the onClose handler.
func (w *WebsocketContext) Close(statusCode websocket.StatusCode, reason string) error {
	err := w.client.Close(statusCode, reason)

	w.closeOnce.Do(w.onClose)

	return errors.Trace(err)
}

// NewWebsocketContext initializes a new websocket connection. Users must
// remember to call WebsocketContext.Close after they are done with the
// connection.
func (wss *WSS) NewWebsocketContext(w http.ResponseWriter, r *http.Request) (*WebsocketContext, error) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		prometheusWSConnErrTotal.Inc()

		w.WriteHeader(http.StatusInternalServerError)

		return nil, errors.Annotatef(err, "accept websocket connection")
	}

	clientID := identifiers.ClientID(path.Base(r.URL.Path))
	room := identifiers.RoomID(path.Base(path.Dir(r.URL.Path)))

	log := wss.log.WithCtx(logger.Ctx{
		"client_id": clientID,
		"room_id":   room,
	})

	log.Info("Enter", nil)
	adapter, _ := wss.rooms.Enter(room)

	client := NewClientWithID(c, clientID)

	log.Info("New websocket connection", nil)

	prometheusWSConnTotal.Inc()
	prometheusWSConnActive.Inc()
	start := time.Now()

	err = adapter.Add(client)
	if multierr.Is(err, ErrDuplicateClientID) {
		client.Close(websocket.StatusPolicyViolation, ErrDuplicateClientID.Error())
		return nil, errors.Annotatef(err, "adapter add - duplicate client id")
	} else if err != nil {
		client.Close(websocket.StatusInternalError, "internal error")
		return nil, errors.Annotatef(err, "adapter add")
	}

	websocketCtx := NewWebsocketContext(adapter, client, room, func() {
		prometheusWSConnActive.Dec()
		duration := time.Since(start)
		prometheusWSConnDuration.Observe(duration.Seconds())

		err := adapter.Remove(clientID)
		if err != nil {
			log.Error("Remove", errors.Trace(err), nil)
		} else {
			log.Info("Remove", nil)
		}

		log.Info("Exit", nil)
		wss.rooms.Exit(room)
	})

	return websocketCtx, nil
}
