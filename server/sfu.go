package server

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/webrtc/v2"
)

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

	allowedInterfaces := map[string]struct{}{}
	for _, iface := range sfuConfig.Interfaces {
		allowedInterfaces[iface] = struct{}{}
	}

	settingEngine := webrtc.SettingEngine{
		LoggerFactory: NewPionLoggerFactory(loggerFactory),
	}
	if len(allowedInterfaces) > 0 {
		settingEngine.SetInterfaceFilter(func(iface string) bool {
			_, ok := allowedInterfaces[iface]
			return ok
		})
	}
	settingEngine.SetTrickle(true)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(webrtc.MediaEngine{}),
		webrtc.WithSettingEngine(settingEngine),
	)

	return &SFU{loggerFactory, wss, iceServers, tracksManager, api}
}

type SFU struct {
	loggerFactory LoggerFactory
	wss           *WSS
	iceServers    []ICEServer
	tracksManager TracksManager
	api           *webrtc.API
}

func (sfu *SFU) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	webrtcICEServers := []webrtc.ICEServer{}
	for _, iceServer := range GetICEAuthServers(sfu.iceServers) {
		var c webrtc.ICECredentialType
		if iceServer.Username != "" && iceServer.Credential != "" {
			c = webrtc.ICECredentialTypePassword
		}
		webrtcICEServers = append(webrtcICEServers, webrtc.ICEServer{
			URLs:           iceServer.URLs,
			CredentialType: c,
			Username:       iceServer.Username,
			Credential:     iceServer.Credential,
		})
	}

	sub, err := sfu.wss.Subscribe(w, r)
	if err != nil {
		log.Printf("Error accepting websocket connection: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	socketHandler := NewSocketHandler(
		sfu.loggerFactory,
		sfu.tracksManager,
		webrtcICEServers,
		sfu.api,
		sub.ClientID,
		sub.Room,
		sub.Adapter,
	)

	for message := range sub.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			log.Printf("[%s] Error handling websocket message: %s", sub.ClientID, err)
		}
	}
	socketHandler.Cleanup()
}

type SocketHandler struct {
	loggerFactory LoggerFactory
	log           Logger
	tracksManager TracksManager
	iceServers    []webrtc.ICEServer
	webrtcAPI     *webrtc.API
	adapter       Adapter
	clientID      string
	room          string

	mu        sync.Mutex
	signaller *Signaller
}

func NewSocketHandler(
	loggerFactory LoggerFactory,
	tracksManager TracksManager,
	iceServers []webrtc.ICEServer,
	webrtcAPI *webrtc.API,
	clientID string,
	room string,
	adapter Adapter,
) *SocketHandler {
	return &SocketHandler{
		loggerFactory: loggerFactory,
		log:           loggerFactory.GetLogger("sfu"),
		tracksManager: tracksManager,
		iceServers:    iceServers,
		webrtcAPI:     webrtcAPI,
		clientID:      clientID,
		room:          room,
		adapter:       adapter,
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

	return fmt.Errorf("Unhandled event: %s", message.Type)
}

func (sh *SocketHandler) Cleanup() {
	if sh.signaller != nil {
		if err := sh.signaller.Close(); err != nil {
			log.Printf("[%s] cleanup: error in signaller.Close: %s", sh.clientID, err)
		}
	}

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", sh.room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		log.Printf("[%s] cleanup: error broadcasting hangUp: %s", sh.clientID, err)
	}
}

func (sh *SocketHandler) handleHangUp(event Message) error {
	clientID := sh.clientID

	log.Printf("[%s] hangUp event", clientID)

	if sh.signaller != nil {
		closeErr := sh.signaller.Close()
		if closeErr != nil {
			return fmt.Errorf("[%s] hangUp: Error closing peer connection: %s", clientID, closeErr)
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

	log.Printf("[%s] Initiator: %s", clientID, initiator)

	if sh.signaller != nil {
		return fmt.Errorf("Unexpected ready event in room %s - already have a signaller", room)
	}

	// FIXME check for errors
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Ready message payload is of wrong type: %T", message.Payload)
	}

	adapter.SetMetadata(clientID, payload["nickname"].(string))

	clients, err := getReadyClients(adapter)
	if err != nil {
		return fmt.Errorf("Error retreiving ready clients: %w", err)
	}

	err = adapter.Broadcast(
		NewMessage("users", room, map[string]interface{}{
			"initiator": initiator,
			"peerIds":   []string{localPeerID},
			"nicknames": clients,
		}),
	)
	if err != nil {
		return fmt.Errorf("Error broadcasting users message: %s", err)
	}

	webrtcConfig := webrtc.Configuration{
		ICEServers: sh.iceServers,
	}

	// Hack to be able to update dynamic codec payload IDs with every new sdp
	// renegotiation of passive (non-server initiated) peer connections.
	field := reflect.ValueOf(sh.webrtcAPI).Elem().FieldByName("mediaEngine")
	unsafeField := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

	mediaEngine, ok := unsafeField.Interface().(*webrtc.MediaEngine)
	if !ok {
		return fmt.Errorf("Error in hack to obtain mediaEngine")
	}

	peerConnection, err := sh.webrtcAPI.NewPeerConnection(webrtcConfig)
	if err != nil {
		return fmt.Errorf("Error creating peer connection: %w", err)
	}
	peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("[%s] ICE gathering state changed: %s", clientID, state)
	})

	closePeer := func(reason error) error {
		err = peerConnection.Close()
		if err != nil {
			return fmt.Errorf("Error closing peer connection: %s. Close was called because: %w", err, reason)
		} else {
			return reason
		}
	}

	var dataChannel *webrtc.DataChannel
	if initiator == localPeerID {
		// need to do this to connect with simple peer
		// only when we are the initiator
		dataChannel, err = peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			return closePeer(fmt.Errorf("Error creating data channel: %w", err))
		}
	}

	// TODO use this to get all client IDs and request all tracks of all users
	// adapter.Clients()
	signaller, err := NewSignaller(
		sh.loggerFactory,
		initiator == localPeerID,
		peerConnection,
		mediaEngine,
		localPeerID,
		clientID,
	)
	if err != nil {
		return closePeer(fmt.Errorf("Error initializing signaller: %w", err))
	}

	sh.signaller = signaller

	prometheusWebRTCConnTotal.Inc()
	prometheusWebRTCConnActive.Inc()

	sh.tracksManager.Add(room, clientID, peerConnection, dataChannel, signaller)
	go sh.processLocalSignals(message, signaller.SignalChannel(), start)
	return nil
}

func (sh *SocketHandler) handleSignal(message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("[%s] Ignoring signal because it is of unexpected type: %T", sh.clientID, payload)
	}

	if sh.signaller == nil {
		return fmt.Errorf("[%s] Ignoring signal '%v' because signaller is not initialized", sh.clientID, payload)
	}

	return sh.signaller.Signal(payload)
}

func (sh *SocketHandler) processLocalSignals(message Message, signals <-chan Payload, startTime time.Time) {
	room := sh.room
	adapter := sh.adapter
	clientID := sh.clientID

	for signal := range signals {
		if _, ok := signal.Signal.(webrtc.SessionDescription); ok {
			if metadata, ok := sh.tracksManager.GetTracksMetadata(clientID); ok {
				err := adapter.Emit(clientID, NewMessage("metadata", room, MetadataPayload{
					UserID:   localPeerID,
					Metadata: metadata,
				}))
				if err != nil {
					log.Printf("[%s] Error sending metadata: %s", clientID, err)
				}
			}
		}
		err := adapter.Emit(clientID, NewMessage("signal", room, signal))
		if err != nil {
			log.Printf("[%s] Error sending local signal: %s", clientID, err)
			// TODO abort connection
		}
	}

	prometheusWebRTCConnActive.Dec()
	prometheusWebRTCConnDuration.Observe(time.Now().Sub(startTime).Seconds())

	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.signaller = nil
	log.Printf("[%s] Peer connection closed, emitting hangUp event", clientID)
	adapter.SetMetadata(clientID, "")

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		log.Printf("[%s] Error broadcasting hangUp: %s", sh.clientID, err)
	}
}
