package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

const IOSH264Fmtp = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"

const localPeerID identifiers.ClientID = "__SERVER__"

const serverIsInitiator = true

type MetadataPayload struct {
	UserID   identifiers.ClientID `json:"userId"`
	Metadata []sfu.TrackMetadata  `json:"metadata"`
}

func NewSFUHandler(
	log logger.Logger,
	wss *WSS,
	iceServers []ICEServer,
	sfuConfig NetworkConfigSFU,
	tracksManager TracksManager,
) *SFU {
	log = log.WithNamespaceAppended("sfu")

	webRTCTransportFactory := NewWebRTCTransportFactory(log, iceServers, sfuConfig)

	return &SFU{log, wss, tracksManager, webRTCTransportFactory}
}

type SFU struct {
	log           logger.Logger
	wss           *WSS
	tracksManager TracksManager

	webRTCTransportFactory *WebRTCTransportFactory
}

func (sfu *SFU) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sub, err := sfu.wss.Subscribe(w, r)
	if err != nil {
		sfu.log.Error("Accept websocket connection", errors.Trace(err), nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log := sfu.log.WithCtx(logger.Ctx{
		"room_id":   sub.Room,
		"client_id": sub.ClientID,
	})

	socketHandler := NewSocketHandler(
		log,
		sfu.tracksManager,
		sfu.webRTCTransportFactory,
		sub.ClientID,
		sub.Room,
		sub.Adapter,
	)

	for message := range sub.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			log.Error("Handle websocket message", errors.Trace(err), nil)
		}
	}

	socketHandler.Cleanup()
}

type SocketHandler struct {
	log                    logger.Logger
	tracksManager          TracksManager
	webRTCTransportFactory *WebRTCTransportFactory
	webRTCTransport        *WebRTCTransport
	adapter                Adapter
	clientID               identifiers.ClientID
	room                   identifiers.RoomID

	mu sync.Mutex
}

func NewSocketHandler(
	log logger.Logger,
	tracksManager TracksManager,
	webRTCTransportFactory *WebRTCTransportFactory,
	clientID identifiers.ClientID,
	room identifiers.RoomID,
	adapter Adapter,
) *SocketHandler {
	return &SocketHandler{
		log:                    log.WithNamespaceAppended("sfu"),
		tracksManager:          tracksManager,
		webRTCTransportFactory: webRTCTransportFactory,
		clientID:               clientID,
		room:                   room,
		adapter:                adapter,
	}
}

func (sh *SocketHandler) HandleMessage(message Message) error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var err error

	switch message.Type {
	case MessageTypeHangUp:
		err = errors.Trace(sh.handleHangUp(message))
	case MessageTypeReady:
		err = errors.Trace(sh.handleReady(message))
	case MessageTypeSignal:
		err = errors.Trace(sh.handleSignal(message))
	case MessageTypeSubTrack:
		err = errors.Trace(sh.handleSubTrackEvent(message))
	case MessageTypePing:
	default:
		err = errors.Errorf("Unhandled event: %s", message.Type)
	}

	return errors.Trace(err)
}

func (sh *SocketHandler) Cleanup() {
	if sh.webRTCTransport != nil {
		if err := sh.webRTCTransport.Close(); err != nil {
			sh.log.Error("Cleanup: close WebRTCTransport", errors.Trace(err), nil)
		}
	}

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", sh.room, map[string]interface{}{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Error("Cleanup: broadcast hangUp", errors.Trace(err), nil)
	}
}

func (sh *SocketHandler) handleSubTrackEvent(m Message) error {
	event := m.Payload.(map[string]interface{})

	// FIXME use strong types.
	pubClientID := event["pubClientId"].(string)
	trackID := identifiers.TrackID(event["trackId"].(string))
	typ := transport.TrackEventType(event["type"].(float64))

	var err error

	switch typ {
	case transport.TrackEventTypeSub:
		err = sh.tracksManager.Sub(sfu.SubParams{
			PubClientID: identifiers.ClientID(pubClientID),
			Room:        identifiers.RoomID(sh.room),
			TrackID:     trackID,
			SubClientID: identifiers.ClientID(sh.clientID),
		})
	case transport.TrackEventTypeUnsub:
		err = sh.tracksManager.Unsub(sfu.SubParams{
			PubClientID: identifiers.ClientID(pubClientID),
			Room:        identifiers.RoomID(sh.room),
			TrackID:     trackID,
			SubClientID: identifiers.ClientID(sh.clientID),
		})
	default:
		err = errors.Errorf("invalid payload type: %d", typ)
	}

	return errors.Trace(err)
}

func (sh *SocketHandler) handleHangUp(_ Message) error {
	clientID := sh.clientID

	sh.log.Info("hangUp event", nil)

	if sh.webRTCTransport != nil {
		err := sh.webRTCTransport.Close()
		if err != nil {
			return errors.Annotatef(err, "hangUp: error closing peer connection for client: %s", clientID)
		}
	}

	return nil
}

func (sh *SocketHandler) handleReady(message Message) error {
	adapter := sh.adapter
	roomID := sh.room
	clientID := sh.clientID
	// userID is the same as clientID for webrtc connections.
	userID := identifiers.UserID(sh.clientID)

	initiator := localPeerID
	if !serverIsInitiator {
		initiator = clientID
	}

	start := time.Now()

	sh.log.Info("ready event", logger.Ctx{
		"initiator": initiator,
	})

	if sh.webRTCTransport != nil {
		return errors.Errorf("unexpected ready event in room %s - already have a webrtc transport", roomID)
	}

	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.Errorf("ready message payload is of wrong type: %T", message.Payload)
	}

	adapter.SetMetadata(clientID, payload["nickname"].(string))

	clients, err := getReadyClients(adapter)
	if err != nil {
		return errors.Annotatef(err, "get ready clients")
	}

	err = adapter.Broadcast(
		NewMessage("users", roomID, map[string]interface{}{
			"initiator": initiator,
			"peerIds":   []identifiers.ClientID{localPeerID},
			"nicknames": clients,
		}),
	)
	if err != nil {
		return errors.Annotatef(err, "broadcasting users")
	}

	webRTCTransport, err := sh.webRTCTransportFactory.NewWebRTCTransport(roomID, clientID, userID)
	if err != nil {
		return errors.Annotatef(err, "create new WebRTCTransport")
	}

	sh.webRTCTransport = webRTCTransport

	prometheusWebRTCConnTotal.Inc()
	prometheusWebRTCConnActive.Inc()

	pubTrackEventsCh, err := sh.tracksManager.Add(roomID, webRTCTransport)
	if err != nil {
		return errors.Trace(err)
	}

	go func() {
		for pubTrackEvent := range pubTrackEventsCh {
			err := sh.adapter.Emit(clientID, Message{
				Type: MessageTypePubTrack,
				Payload: map[string]interface{}{
					"trackId":     pubTrackEvent.PubTrack.TrackID,
					"pubClientId": pubTrackEvent.PubTrack.ClientID,
					"userId":      pubTrackEvent.PubTrack.UserID,
					"type":        pubTrackEvent.Type,
				},
				Room: roomID,
			})
			if err != nil {
				sh.log.Error("Emit pub track event", errors.Trace(err), nil)
			}
		}
	}()

	go sh.processLocalSignals(message, webRTCTransport.SignalChannel(), start)

	return nil
}

func (sh *SocketHandler) handleSignal(message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.Errorf("signal: unexpected type %T", payload)
	}

	if sh.webRTCTransport == nil {
		return errors.Errorf("signal: webRTCTransport not initialized")
	}

	err := sh.webRTCTransport.Signal(payload)
	return errors.Annotate(err, "handleSignal")
}

func (sh *SocketHandler) processLocalSignals(_ Message, signals <-chan Payload, startTime time.Time) {
	room := sh.room
	adapter := sh.adapter
	clientID := sh.clientID

	for signal := range signals {
		if _, ok := signal.Signal.(webrtc.SessionDescription); ok {
			if metadata, ok := sh.tracksManager.TracksMetadata(room, clientID); ok {
				err := adapter.Emit(clientID, NewMessage("metadata", room, MetadataPayload{
					UserID:   localPeerID,
					Metadata: metadata,
				}))
				if err != nil {
					sh.log.Error("Send metadata", errors.Trace(err), nil)
				}
			}
		}

		err := adapter.Emit(clientID, NewMessage("signal", room, signal))
		if err != nil {
			sh.log.Error("Send local signal", errors.Trace(err), nil)
			// TODO abort connection
		}
	}

	prometheusWebRTCConnActive.Dec()
	prometheusWebRTCConnDuration.Observe(time.Since(startTime).Seconds())

	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.webRTCTransport = nil
	sh.log.Info("Peer connection closed, send hangUp event", nil)
	adapter.SetMetadata(clientID, "")

	err := sh.adapter.Broadcast(
		// FIXME strong types.
		NewMessage("hangUp", room, map[string]identifiers.ClientID{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Error("Broadcast hangUp", errors.Trace(err), nil)
	}
}
