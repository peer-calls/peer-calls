package routes

import (
	"net/http"
	"time"

	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
	"github.com/jeremija/peer-calls/src/server-go/wrtc"
	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/pion/webrtc/v2"
)

const (
	rtcpPLIInterval = time.Second * 3
)

var peers = map[string]string{}

const localPeerID = "__SERVER__"

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

		// peerConnection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		// 	// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// 	// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
		// 	go func() {
		// 		ticker := time.NewTicker(rtcpPLIInterval)
		// 		for range ticker.C {
		// 			if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}}); rtcpSendErr != nil {
		// 				log.Printf("Error sending rtcp PLI: %s", rtcpSendErr)
		// 			}
		// 		}
		// 	}()

		// 	// Create a local track, all our SFU clients will be fed via this track
		// 	localTrack, newTrackErr := peerConnection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
		// 	if newTrackErr != nil {
		// 		panic(newTrackErr)
		// 	}

		// 	// FIXME notify other peers in same room that we got a new track!
		// 	// localTrackChan <- localTrack

		// 	rtpBuf := make([]byte, 1400)
		// 	for {
		// 		i, readErr := remoteTrack.Read(rtpBuf)
		// 		if readErr != nil {
		// 			panic(readErr)
		// 		}

		// 		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		// 		if _, err = localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
		// 			panic(err)
		// 		}
		// 	}
		// })

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
