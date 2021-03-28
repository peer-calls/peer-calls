package server

import (
	"fmt"
	"net/http"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/message"
)

type ReadyMessage struct {
	UserID string `json:"userId"`
	Room   string `json:"room"`
}

func NewMeshHandler(log logger.Logger, wss *WSS) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log = log.WithNamespaceAppended("mesh")

		sub, err := wss.Subscribe(w, r)
		if err != nil {
			log.Error("Subscribe to websocket", errors.Trace(err), nil)
		}

		for msg := range sub.Messages {
			adapter := sub.Adapter
			room := sub.Room
			clientID := sub.ClientID

			log = log.WithCtx(logger.Ctx{
				"client_id": clientID,
				"room_id":   room,
			})

			var (
				err error
			)

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
					message.NewUsers(room, message.Users{
						Initiator: clientID,
						PeerIDs:   clientsToPeerIDs(clients),
						Nicknames: clients,
					}),
				)
				err = errors.Annotatef(err, "ready broadcast")
			case message.TypeSignal:
				signal := *msg.Payload.Signal

				targetClientID := signal.UserID

				log.Info("Send signal to", logger.Ctx{
					"target_client_id": targetClientID,
				})
				err = adapter.Emit(targetClientID, message.NewSignal(room, message.UserSignal{
					Signal: signal.Signal,
					UserID: clientID,
				}))
				err = errors.Annotatef(err, "signal emit")
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
