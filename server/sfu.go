package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/pion/webrtc/v3"
)

const IOSH264Fmtp = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"

const localPeerID = "__SERVER__"

const serverIsInitiator = true

type MetadataPayload struct {
	UserID   string          `json:"userId"`
	Metadata []TrackMetadata `json:"metadata"`
}

func NewSFUHandler(
	loggerFactory LoggerFactory,
	wss *WSS,
	iceServers []ICEServer,
	sfuConfig NetworkConfigSFU,
	tracksManager TracksManager,
) *SFU {
	log := loggerFactory.GetLogger("sfu")

	webRTCTransportFactory := NewWebRTCTransportFactory(loggerFactory, iceServers, sfuConfig)

	return &SFU{loggerFactory, log, wss, tracksManager, webRTCTransportFactory}
}

type SFU struct {
	loggerFactory LoggerFactory
	log           Logger
	wss           *WSS
	tracksManager TracksManager

	webRTCTransportFactory *WebRTCTransportFactory
}

func (sfu *SFU) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sub, err := sfu.wss.Subscribe(w, r)
	if err != nil {
		sfu.log.Printf("Error accepting websocket connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	socketHandler := NewSocketHandler(
		sfu.loggerFactory,
		sfu.tracksManager,
		sfu.webRTCTransportFactory,
		sub.ClientID,
		sub.Room,
		sub.Adapter,
	)

	for message := range sub.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			sfu.log.Printf("[%s] Error handling websocket message: %s", sub.ClientID, err)
		}
	}

	socketHandler.Cleanup()
}

type SocketHandler struct {
	loggerFactory          LoggerFactory
	log                    Logger
	tracksManager          TracksManager
	webRTCTransportFactory *WebRTCTransportFactory
	webRTCTransport        *WebRTCTransport
	adapter                Adapter
	clientID               string
	room                   string

	mu sync.Mutex
}

func NewSocketHandler(
	loggerFactory LoggerFactory,
	tracksManager TracksManager,
	webRTCTransportFactory *WebRTCTransportFactory,
	clientID string,
	room string,
	adapter Adapter,
) *SocketHandler {
	return &SocketHandler{
		loggerFactory:          loggerFactory,
		log:                    loggerFactory.GetLogger("sfu"),
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

	switch message.Type {
	case "hangUp":
		return sh.handleHangUp(message)
	case "ready":
		return sh.handleReady(message)
	case "signal":
		return sh.handleSignal(message)
	case "ping":
		return nil
	}

	return errors.Errorf("Unhandled event: %s", message.Type)
}

func (sh *SocketHandler) Cleanup() {
	if sh.webRTCTransport != nil {
		if err := sh.webRTCTransport.Close(); err != nil {
			sh.log.Printf("[%s] cleanup: error in webRTCTransport.Close: %s", sh.clientID, err)
		}
	}

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", sh.room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Printf("[%s] cleanup: error broadcasting hangUp: %s", sh.clientID, err)
	}
}

func (sh *SocketHandler) handleHangUp(_ Message) error {
	clientID := sh.clientID

	sh.log.Printf("[%s] hangUp event", clientID)

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
	room := sh.room
	clientID := sh.clientID

	initiator := localPeerID
	if !serverIsInitiator {
		initiator = clientID
	}

	start := time.Now()

	sh.log.Printf("[%s] Initiator: %s", clientID, initiator)

	if sh.webRTCTransport != nil {
		return errors.Errorf("unexpected ready event in room %s - already have a webrtc transport", room)
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
		NewMessage("users", room, map[string]interface{}{
			"initiator": initiator,
			"peerIds":   []string{localPeerID},
			"nicknames": clients,
		}),
	)
	if err != nil {
		return errors.Annotatef(err, "broadcasting users")
	}

	webRTCTransport, err := sh.webRTCTransportFactory.NewWebRTCTransport(clientID)
	if err != nil {
		return errors.Annotatef(err, "create new WebRTCTransport")
	}

	sh.webRTCTransport = webRTCTransport

	prometheusWebRTCConnTotal.Inc()
	prometheusWebRTCConnActive.Inc()

	sh.tracksManager.Add(room, webRTCTransport)

	go sh.processLocalSignals(message, webRTCTransport.SignalChannel(), start)

	return nil
}

func (sh *SocketHandler) handleSignal(message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.Errorf("[%s] Ignoring signal because it is of unexpected type: %T", sh.clientID, payload)
	}

	if sh.webRTCTransport == nil {
		return errors.Errorf("[%s] Ignoring signal '%v' because webRTCTransport is not initialized", sh.clientID, payload)
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
			if metadata, ok := sh.tracksManager.GetTracksMetadata(room, clientID); ok {
				err := adapter.Emit(clientID, NewMessage("metadata", room, MetadataPayload{
					UserID:   localPeerID,
					Metadata: metadata,
				}))
				if err != nil {
					sh.log.Printf("[%s] Error sending metadata: %+v", clientID, errors.Trace(err))
				}
			}
		}

		err := adapter.Emit(clientID, NewMessage("signal", room, signal))
		if err != nil {
			sh.log.Printf("[%s] Error sending local signal: %+v", clientID, errors.Trace(err))
			// TODO abort connection
		}
	}

	prometheusWebRTCConnActive.Dec()
	prometheusWebRTCConnDuration.Observe(time.Since(startTime).Seconds())

	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.webRTCTransport = nil
	sh.log.Printf("[%s] Peer connection closed, emitting hangUp event", clientID)
	adapter.SetMetadata(clientID, "")

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Printf("[%s] Error broadcasting hangUp: %+v", sh.clientID, errors.Trace(err))
	}
}
