package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"nhooyr.io/websocket"
)

type ReadyMessage struct {
	PeerID string `json:"peerId"`
	Room   string `json:"room"`
}

func NewMeshHandler(log logger.Logger, wss *WSS) http.Handler {
	log = log.WithNamespaceAppended("mesh")

	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		websocketCtx, err := wss.NewWebsocketContext(w, r)
		if err != nil {
			log.Error("Create websocket context", errors.Trace(err), nil)
			return
		}

		roomID := websocketCtx.RoomID()
		clientID := websocketCtx.ClientID()

		// Just in case. I'm actually not sure if this is necessary since if the
		// reading stops, it most likely means the connection has already been
		// closed.
		defer websocketCtx.Close(websocket.StatusNormalClosure, "")

		adapter := websocketCtx.Adapter()

		pinger := NewPinger(ctx, 5*time.Second, func() {
			adapter.Emit(clientID, message.NewPing(roomID))
		})

		for msg := range websocketCtx.Messages() {
			log = log.WithCtx(logger.Ctx{
				"client_id": clientID,
				"room_id":   roomID,
			})

			var err error

			switch msg.Type {
			case message.TypeHangUp:
				log.Info("hangUp event", nil)
				adapter.SetMetadata(clientID, "")
			case message.TypeReady:
				ready := *msg.Payload.Ready
				adapter.SetMetadata(clientID, ready.Nickname)

				clients, readyClientsErr := getReadyClients(adapter)
				if readyClientsErr != nil {
					log.Error("Retrieve clients", errors.Trace(err), nil)
				}

				log.Info(fmt.Sprintf("Got clients: %s", clients), nil)

				err = adapter.Broadcast(
					message.NewUsers(roomID, message.Users{
						Initiator: clientID,
						PeerIDs:   clientsToPeerIDs(clients),
						Nicknames: clients,
					}),
				)
				err = errors.Annotatef(err, "ready broadcast")
			case message.TypeSignal:
				signal := *msg.Payload.Signal

				targetClientID := signal.PeerID

				log.Info("Send signal to", logger.Ctx{
					"target_client_id": targetClientID,
				})
				err = adapter.Emit(targetClientID, message.NewSignal(roomID, message.UserSignal{
					Signal: signal.Signal,
					PeerID: clientID,
				}))
				err = errors.Annotatef(err, "signal emit")
			case message.TypePong:
				pinger.ReceivePong()
			}

			if err != nil {
				log.Error("Send event", errors.Trace(err), nil)
			}
		}
	}

	return http.HandlerFunc(fn)
}

func getReadyClients(adapter Adapter) (map[identifiers.ClientID]string, error) {
	filteredClients := map[identifiers.ClientID]string{}
	clients, err := adapter.Clients()
	if err != nil {
		return filteredClients, errors.Annotate(err, "ready clients")
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

func clientsToPeerIDs(clients map[identifiers.ClientID]string) (peers []identifiers.ClientID) {
	for clientID := range clients {
		peers = append(peers, clientID)
	}
	return
}
