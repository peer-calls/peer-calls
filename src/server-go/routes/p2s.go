package routes

import (
	"net/http"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
	"github.com/jeremija/peer-calls/src/server-go/wrtc"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/tracks"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/pion/webrtc/v2"
)

const localPeerID = "__SERVER__"

type TracksManager interface {
	Add(room string, clientID string, peerConnection tracks.PeerConnection)
}

func NewPeerToServerRoomHandler(
	wss *wsserver.WSS,
	iceServers []config.ICEServer,
	tracksManager TracksManager,
) http.Handler {

	webrtcICEServers := []webrtc.ICEServer{}
	for _, iceServer := range iceServers {
		var c webrtc.ICECredentialType
		if iceServer.AuthType == config.AuthTypeSecret {
			c = webrtc.ICECredentialTypePassword
		}
		webrtcICEServers = append(webrtcICEServers, webrtc.ICEServer{
			URLs:           iceServer.URLs,
			CredentialType: c,
			Username:       iceServer.AuthSecret.Username,
			Credential:     iceServer.AuthSecret.Secret,
		})
	}

	webrtcConfig := webrtc.Configuration{
		ICEServers: webrtcICEServers,
	}

	fn := func(w http.ResponseWriter, r *http.Request) {

		mediaEngine := webrtc.MediaEngine{}
		settingEngine := webrtc.SettingEngine{}
		settingEngine.SetTrickle(true)
		api := webrtc.NewAPI(
			webrtc.WithMediaEngine(mediaEngine),
			webrtc.WithSettingEngine(settingEngine),
		)
		peerConnection, err := api.NewPeerConnection(webrtcConfig)
		if err != nil {
			log.Printf("Error creating peer connection: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			log.Printf("Peer connection state changed %s", connectionState.String())
		})

		// make chan

		// peerConnection, err := api.NewPeerConnection(config)

		cleanup := func() {
			// TODO maybe cleanup is not necessary as we can still keep peer
			// connections after websocket conn closes
		}

		var signaller *wrtc.Signaller

		handleMessage := func(event wsserver.RoomEvent) {
			msg := event.Message
			adapter := event.Adapter
			room := event.Room
			clientID := event.ClientID

			var responseEventName string
			var err error

			switch msg.Type {
			case "ready":
				responseEventName = "users"
				err = adapter.Broadcast(
					wsmessage.NewMessage(responseEventName, room, map[string]interface{}{
						"initiator": clientID,
						"users":     []User{{UserID: localPeerID, ClientID: localPeerID}},
					}),
				)
				// TODO use this to get all client IDs and request all tracks of all users
				// adapter.Clients()
				if signaller == nil {
					signaller = wrtc.NewSignaller(
						peerConnection,
						localPeerID,
						func(signal wrtc.SignalSDP) error {
							return adapter.Emit(clientID, wsmessage.NewMessage("signal", room, signal))
						},
						func(signal wrtc.SignalCandidate) {
							err := adapter.Emit(clientID, wsmessage.NewMessage("signal", room, signal))
							if err != nil {
								log.Printf("Error emitting signal candidate to clientID: %s: %s", clientID, err)
							}
						},
					)
					tracksManager.Add(room, clientID, peerConnection)
				}
			case "signal":
				payload, _ := msg.Payload.(map[string]interface{})
				err = signaller.Signal(payload)
			}

			if err != nil {
				log.Printf("Error handling event (event: %s, room: %s, source: %s): %s", msg.Type, room, clientID, err)
			}
		}

		wss.HandleRoomWithCleanup(w, r, handleMessage, cleanup)
	}
	return http.HandlerFunc(fn)
}
