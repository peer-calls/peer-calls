package routes

import (
	"fmt"
	"net/http"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/pion/webrtc/v2"
)

var peers = map[string]string{}

const serverUserID = "__SERVER__"

func NewPeerToServerRoomHandler(
	wss *wsserver.WSS,
	iceServers []config.ICEServer,
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

		startedTrickle := false
		startTrickle := func(readyEvent wsserver.RoomEvent) {
			adapter := readyEvent.Adapter
			room := readyEvent.Room
			clientID := readyEvent.ClientID

			if startedTrickle {
				return
			}
			startedTrickle = true
			peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}

				candidate := c.ToJSON()

				log.Printf("Got ice candidate from sever peer: %s", candidate)

				adapter.Emit(clientID, wsmessage.NewMessage("signal", room, map[string]interface{}{
					"userId": serverUserID,
					"signal": map[string]interface{}{
						"candidate": map[string]interface{}{
							"candidate":     candidate.Candidate,
							"sdpMLineIndex": candidate.SDPMLineIndex,
							// "sdpMid":        candidate.SDPMid,
						},
					},
				}))
			})
		}

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
						"users":     []User{{UserID: serverUserID, ClientID: serverUserID}},
					}),
				)
				startTrickle(event)
			case "signal":
				payload, _ := msg.Payload.(map[string]interface{})
				signal, _ := payload["signal"].(map[string]interface{})
				targetClientID, _ := payload["userId"].(string)

				if targetClientID != serverUserID {
					// this is a hack
					err = fmt.Errorf("Peer2Server only sends signal to server as peer")
					break
				}

				if candidate, ok := signal["candidate"]; ok {
					log.Printf("Got client ice candidate: %s", candidate)
					if candidateString, ok := candidate.(string); ok {
						iceCandidate := webrtc.ICECandidateInit{Candidate: candidateString}
						err = peerConnection.AddICECandidate(iceCandidate)
					}
				} else if sdpTypeString, ok := signal["type"]; ok {
					sdpString, _ := signal["sdp"].(string)
					sdp := webrtc.SessionDescription{}
					sdp.SDP = sdpString
					log.Printf("Got client signal: %s", sdp)
					switch sdpTypeString {
					case "offer":
						sdp.Type = webrtc.SDPTypeOffer
						mediaEngine.PopulateFromSDP(sdp)
						// videoCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
						// audioCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)
					case "answer":
						sdp.Type = webrtc.SDPTypeAnswer
					case "pranswer":
						sdp.Type = webrtc.SDPTypePranswer
					case "rollback":
						sdp.Type = webrtc.SDPTypeRollback
					default:
						err = fmt.Errorf("Unknown sdp type: %s", sdpString)
					}

					if err != nil {
						break
					}

					if err2 := peerConnection.SetRemoteDescription(sdp); err2 != nil {
						err = fmt.Errorf("Error setting remote description: %w", err2)
						break
					}
					answer, err2 := peerConnection.CreateAnswer(nil)
					if err2 != nil {
						err = fmt.Errorf("Error creating answer: %w", err2)
						break
					}
					if err2 := peerConnection.SetLocalDescription(answer); err2 != nil {
						err = fmt.Errorf("Error setting local description: %w", err)
					}
					log.Printf("Emitting answer: %s", answer)
					err = adapter.Emit(event.ClientID, wsmessage.NewMessage("signal", room, map[string]interface{}{
						"userId": serverUserID,
						"signal": answer,
					}))
				} else {
					err = fmt.Errorf("Unexpected signal message")
				}

			}

			if err != nil {
				log.Printf("Error handling event (event: %s, room: %s, source: %s): %s", msg.Type, room, clientID, err)
			}
		}

		wss.HandleRoomWithCleanup(w, r, handleMessage, cleanup)
	}
	return http.HandlerFunc(fn)
}
