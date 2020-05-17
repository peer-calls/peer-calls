package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v2"
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
	log.Printf("Registering media engine codecs")
	var mediaEngine webrtc.MediaEngine
	RegisterCodecs(&mediaEngine)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	return &SFU{loggerFactory, log, wss, iceServers, tracksManager, api}
}

func RegisterCodecs(mediaEngine *webrtc.MediaEngine) {
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		// webrtc.RTCPFeedback{
		// 	Type: webrtc.TypeRTCPFBGoogREMB,
		// },
		// webrtc.RTCPFeedback{
		// 	Type:      webrtc.TypeRTCPFBCCM,
		// 	Parameter: "fir",
		// },
		// webrtc.RTCPFeedback{
		// 	Type: webrtc.TypeRTCPFBNACK,
		// },
		webrtc.RTCPFeedback{
			Type:      webrtc.TypeRTCPFBNACK,
			Parameter: "pli",
		},
	}

	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
	// s.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
	// s.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
}

type SFU struct {
	loggerFactory LoggerFactory
	log           Logger
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
		sfu.log.Printf("Error accepting websocket connection: %s", err)
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
			sfu.log.Printf("[%s] Error handling websocket message: %s", sub.ClientID, err)
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
			sh.log.Printf("[%s] cleanup: error in signaller.Close: %s", sh.clientID, err)
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

func (sh *SocketHandler) handleHangUp(event Message) error {
	clientID := sh.clientID

	sh.log.Printf("[%s] hangUp event", clientID)

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

	sh.log.Printf("[%s] Initiator: %s", clientID, initiator)

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

	peerConnection, err := sh.webrtcAPI.NewPeerConnection(webrtcConfig)
	if err != nil {
		return fmt.Errorf("Error creating peer connection: %w", err)
	}
	peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		sh.log.Printf("[%s] ICE gathering state changed: %s", clientID, state)
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
			if metadata, ok := sh.tracksManager.GetTracksMetadata(room, clientID); ok {
				err := adapter.Emit(clientID, NewMessage("metadata", room, MetadataPayload{
					UserID:   localPeerID,
					Metadata: metadata,
				}))
				if err != nil {
					sh.log.Printf("[%s] Error sending metadata: %s", clientID, err)
				}
			}
		}
		err := adapter.Emit(clientID, NewMessage("signal", room, signal))
		if err != nil {
			sh.log.Printf("[%s] Error sending local signal: %s", clientID, err)
			// TODO abort connection
		}
	}

	prometheusWebRTCConnActive.Dec()
	prometheusWebRTCConnDuration.Observe(time.Now().Sub(startTime).Seconds())

	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.signaller = nil
	sh.log.Printf("[%s] Peer connection closed, emitting hangUp event", clientID)
	adapter.SetMetadata(clientID, "")

	err := sh.adapter.Broadcast(
		NewMessage("hangUp", room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Printf("[%s] Error broadcasting hangUp: %s", sh.clientID, err)
	}
}
