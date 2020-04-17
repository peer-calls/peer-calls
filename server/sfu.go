package server

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"unsafe"

	"github.com/pion/logging"
	"github.com/pion/webrtc/v2"
)

const localPeerID = "__SERVER__"

type pionLogger struct {
	traceLogger Logger
	debugLogger Logger
	infoLogger  Logger
	warnLogger  Logger
	errorLogger Logger
}

type pionLoggerFactory struct {
	loggerFactory LoggerFactory
}

func newPionLoggerFactory(loggerFactory LoggerFactory) *pionLoggerFactory {
	return &pionLoggerFactory{loggerFactory}
}

func (p pionLoggerFactory) NewLogger(subsystem string) logging.LeveledLogger {
	return &pionLogger{
		traceLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":trace"),
		debugLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":debug"),
		infoLogger:  p.loggerFactory.GetLogger("pion:" + subsystem + ":info"),
		warnLogger:  p.loggerFactory.GetLogger("pion:" + subsystem + ":warn"),
		errorLogger: p.loggerFactory.GetLogger("pion:" + subsystem + ":error"),
	}
}

func (p *pionLogger) Trace(msg string) {
	p.traceLogger.Println(msg)
}
func (p *pionLogger) Tracef(format string, args ...interface{}) {
	p.traceLogger.Printf(format, args...)
}
func (p *pionLogger) Debug(msg string) {
	p.debugLogger.Println(msg)
}
func (p *pionLogger) Debugf(format string, args ...interface{}) {
	p.debugLogger.Printf(format, args...)
}
func (p *pionLogger) Info(msg string) {
	p.infoLogger.Println(msg)
}
func (p *pionLogger) Infof(format string, args ...interface{}) {
	p.infoLogger.Printf(format, args...)
}
func (p *pionLogger) Warn(msg string) {
	p.warnLogger.Println(msg)
}
func (p *pionLogger) Warnf(format string, args ...interface{}) {
	p.warnLogger.Printf(format, args...)
}
func (p *pionLogger) Error(msg string) {
	p.errorLogger.Println(msg)
}
func (p *pionLogger) Errorf(format string, args ...interface{}) {
	p.errorLogger.Printf(format, args...)
}

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
) http.Handler {
	log := loggerFactory.GetLogger("sfu")

	fn := func(w http.ResponseWriter, r *http.Request) {

		webrtcICEServers := []webrtc.ICEServer{}
		for _, iceServer := range GetICEAuthServers(iceServers) {
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

		webrtcConfig := webrtc.Configuration{
			ICEServers: webrtcICEServers,
		}

		allowedInterfaces := map[string]struct{}{}
		for _, iface := range sfuConfig.Interfaces {
			allowedInterfaces[iface] = struct{}{}
		}

		settingEngine := webrtc.SettingEngine{
			LoggerFactory: newPionLoggerFactory(loggerFactory),
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

		// Hack to be able to update dynamic codec payload IDs with every new sdp
		// renegotiation of passive (non-server initiated) peer connections.
		field := reflect.ValueOf(api).Elem().FieldByName("mediaEngine")
		unsafeField := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		mediaEngine, ok := unsafeField.Interface().(*webrtc.MediaEngine)
		if !ok {
			log.Printf("Error in hack to obtain mediaEngine")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var signaller *Signaller
		var signallerMu sync.Mutex

		cleanup := func(event CleanupEvent) {
			signallerMu.Lock()
			defer signallerMu.Unlock()

			if signaller != nil {
				if err := signaller.Close(); err != nil {
					log.Printf("[%s] cleanup: error in signaller.Close: %s", event.ClientID, err)
				}
			}

			err := event.Adapter.Broadcast(
				NewMessage("hangUp", event.Room, map[string]string{
					"userId": event.ClientID,
				}),
			)
			if err != nil {
				log.Printf("[%s] cleanup: error broadcasting hangUp: %s", event.ClientID, err)
			}
		}

		handleMessage := func(event RoomEvent) {
			// log.Printf("[%s] got message, %s", event.ClientID, event.Message.Type)
			signallerMu.Lock()
			defer signallerMu.Unlock()

			msg := event.Message
			adapter := event.Adapter
			room := event.Room
			clientID := event.ClientID

			initiator := localPeerID
			if !serverIsInitiator {
				initiator = clientID
			}

			var err error

			switch msg.Type {
			case "hangUp":
				log.Printf("[%s] hangUp event", clientID)
				if signaller != nil {
					closeErr := signaller.Close()
					if closeErr != nil {
						err = fmt.Errorf("[%s] hangUp: Error closing peer connection: %s", clientID, closeErr)
					}
				}
			case "ready":
				log.Printf("[%s] Initiator: %s", clientID, initiator)

				peerConnection, pcErr := api.NewPeerConnection(webrtcConfig)
				if pcErr != nil {
					err = fmt.Errorf("[%s] Error creating peer connection: %s", clientID, pcErr)
					break
				}
				peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
					log.Printf("[%s] ICE gathering state changed: %s", clientID, state)
				})

				// FIXME check for errors
				payload, _ := msg.Payload.(map[string]interface{})
				adapter.SetMetadata(clientID, payload["nickname"].(string))

				clients, clientsError := getReadyClients(adapter)
				if clientsError != nil {
					log.Printf("[%s] Error retrieving clients: %s", clientID, err)
				}

				broadcastErr := adapter.Broadcast(
					NewMessage("users", room, map[string]interface{}{
						"initiator": initiator,
						"peerIds":   []string{localPeerID},
						"nicknames": clients,
					}),
				)
				if broadcastErr != nil {
					log.Printf("[%s] Error broadcasting users message: %s", clientID, err)
					break
				}

				var dataChannel *webrtc.DataChannel
				if initiator == localPeerID {
					// need to do this to connect with simple peer
					// only when we are the initiator
					dataChannel, err = peerConnection.CreateDataChannel("data", nil)
					if err != nil {
						log.Printf("[%s] Error creating data channel: %s", clientID, err)
						// TODO abort connection
					}
				}

				// TODO use this to get all client IDs and request all tracks of all users
				// adapter.Clients()
				if signaller == nil {
					signaller, err = NewSignaller(
						loggerFactory,
						initiator == localPeerID,
						peerConnection,
						mediaEngine,
						localPeerID,
						clientID,
					)
					if err != nil {
						err = fmt.Errorf("[%s] Error initializing signaller: %s", clientID, err)
						break
					}
					signalChannel := signaller.SignalChannel()
					tracksManager.Add(room, clientID, peerConnection, dataChannel, signaller)
					go func() {
						for signal := range signalChannel {
							if _, ok := signal.Signal.(webrtc.SessionDescription); ok {
								if metadata, ok := tracksManager.GetTracksMetadata(clientID); ok {
									adapter.Emit(clientID, NewMessage("metadata", room, MetadataPayload{
										UserID:   localPeerID,
										Metadata: metadata,
									}))
								}
							}
							err := adapter.Emit(clientID, NewMessage("signal", room, signal))
							if err != nil {
								log.Printf("[%s] Error sending local signal: %s", clientID, err)
								// TODO abort connection
							}
						}

						signallerMu.Lock()
						defer signallerMu.Unlock()
						signaller = nil
						log.Printf("[%s] Peer connection closed, emitting hangUp event", clientID)
						adapter.SetMetadata(clientID, "")

						err := event.Adapter.Broadcast(
							NewMessage("hangUp", event.Room, map[string]string{
								"userId": event.ClientID,
							}),
						)
						if err != nil {
							log.Printf("[%s] Error brodacastin hangUp: %s", event.ClientID, err)
						}
						return
					}()
				}
			case "signal":
				payload, _ := msg.Payload.(map[string]interface{})
				if signaller == nil {
					err = fmt.Errorf("[%s] Ignoring signal because signaller is not initialized", clientID)
				} else {
					err = signaller.Signal(payload)
				}
			}

			if err != nil {
				log.Printf("[%s] Error handling event (event: %s, room: %s): %s", clientID, msg.Type, room, err)
			}
		}

		wss.HandleRoomWithCleanup(w, r, handleMessage, cleanup)
	}
	return http.HandlerFunc(fn)
}
