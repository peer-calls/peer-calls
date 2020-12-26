package server

import (
	"fmt"
	"net/http"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
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
			log.Error(errors.Annotate(err, "subscribe to websocket messages"), nil)
		}

		for msg := range sub.Messages {
			adapter := sub.Adapter
			room := sub.Room
			clientID := sub.ClientID

			log = log.WithCtx(logger.Ctx{
				"client_id": clientID,
			})

			var (
				responseEventName MessageType
				err               error
			)

			switch msg.Type {
			case MessageTypeHangUp:
				log.Info("hangUp event", nil)
				adapter.SetMetadata(clientID, "")
			case MessageTypeReady:
				// FIXME check for errors
				payload, _ := msg.Payload.(map[string]interface{})
				adapter.SetMetadata(clientID, payload["nickname"].(string))

				clients, readyClientsErr := getReadyClients(adapter)
				if readyClientsErr != nil {
					log.Error(errors.Annotate(err, "retrieving clients"), nil)
				}

				log.Info(fmt.Sprintf("Got clients: %s", clients), nil)

				err = adapter.Broadcast(
					NewMessage(MessageTypeUsers, room, map[string]interface{}{
						"initiator": clientID,
						"peerIds":   clientsToPeerIDs(clients),
						"nicknames": clients,
					}),
				)
				err = errors.Annotatef(err, "ready broadcast")
			case MessageTypeSignal:
				payload, _ := msg.Payload.(map[string]interface{})
				signal := payload["signal"]
				targetClientID, _ := payload["userId"].(string)

				log.Info("Send signal to", logger.Ctx{
					"target_client_id": targetClientID,
				})
				err = adapter.Emit(targetClientID, NewMessage(MessageTypeSignal, room, map[string]interface{}{
					"userId": clientID,
					"signal": signal,
				}))
				err = errors.Annotatef(err, "signal emit")
			}

			if err != nil {
				log.Error(errors.Annotate(err, "sending event"), logger.Ctx{
					"room":       room,
					"event_name": responseEventName,
				})
			}
		}
	}
	return http.HandlerFunc(fn)
}

func getReadyClients(adapter Adapter) (map[string]string, error) {
	filteredClients := map[string]string{}
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

func clientsToPeerIDs(clients map[string]string) (peers []string) {
	for clientID := range clients {
		peers = append(peers, clientID)
	}
	return
}
