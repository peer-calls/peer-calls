package message_test

import (
	"encoding/json"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func TestMessage_JSON(t *testing.T) {
	messages := []message.Message{
		{
			Type: message.TypeHangUp,
			Room: "test",
			Payload: message.Payload{
				HangUp: &message.HangUp{
					PeerID: identifiers.ClientID("test"),
				},
			},
		},
		{
			Type: message.TypeReady,
			Room: "test",
			Payload: message.Payload{
				Ready: &message.Ready{
					Nickname: "nick",
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.UserSignal{
					PeerID: identifiers.ClientID("client123"),
					Signal: message.Signal{
						Candidate: &webrtc.ICECandidateInit{
							Candidate: "a",
							SDPMLineIndex: func() *uint16 {
								val := uint16(1)
								return &val
							}(),
							SDPMid: func() *string {
								val := "c"
								return &val
							}(),
						},
					},
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.UserSignal{
					PeerID: identifiers.ClientID("client123"),
					Signal: message.Signal{
						Type: message.SignalTypeTransceiverRequest,
						TransceiverRequest: &message.TransceiverRequest{
							Kind: transport.TrackKindAudio,
							Init: message.TransceiverInit{
								Direction: message.DirectionSendRecv,
							},
						},
					},
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.UserSignal{
					PeerID: identifiers.ClientID("client123"),
					Signal: message.Signal{
						Type:        message.SignalTypeRenegotiate,
						Renegotiate: true,
					},
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.UserSignal{
					PeerID: identifiers.ClientID("client123"),
					Signal: message.Signal{
						Type: message.SignalTypeOffer,
						SDP:  "-sdp-",
					},
				},
			},
		},
		{
			Type: message.TypePing,
			Room: "test",
			Payload: message.Payload{
				Ping: &message.Ping{},
			},
		},
		{
			Type: message.TypePubTrack,
			Room: "test",
			Payload: message.Payload{
				PubTrack: &message.PubTrack{
					TrackID:     identifiers.TrackID{ID: "123", StreamID: "456"},
					PubClientID: identifiers.ClientID("client123"),
					PeerID:      identifiers.PeerID("user123"),
					Kind:        transport.TrackKindVideo,
					Type:        transport.TrackEventTypeAdd,
				},
			},
		},
		{
			Type: message.TypeSubTrack,
			Room: "test",
			Payload: message.Payload{
				SubTrack: &message.SubTrack{
					TrackID:     identifiers.TrackID{ID: "123", StreamID: "456"},
					PubClientID: identifiers.ClientID("client123"),
					Type:        transport.TrackEventTypeAdd,
				},
			},
		},
		{
			Type: message.TypeRoomJoin,
			Room: "test",
			Payload: message.Payload{
				RoomJoin: &message.RoomJoin{
					ClientID: identifiers.ClientID("user123"),
					Metadata: "{}",
				},
			},
		},
		{
			Type: message.TypeRoomLeave,
			Room: "test",
			Payload: message.Payload{
				RoomLeave: "user123",
			},
		},
		{
			Type: message.TypeUsers,
			Room: "test",
			Payload: message.Payload{
				Users: &message.Users{
					Initiator: "user123",
					PeerIDs:   []identifiers.ClientID{"client123"},
					Nicknames: map[identifiers.ClientID]string{
						"clinet444": "four-four-four",
					},
				},
			},
		},
	}

	for _, m := range messages {
		b, err := json.Marshal(m)
		assert.NoError(t, err, "marshal message: %+v", m)

		var m2 message.Message

		err = json.Unmarshal(b, &m2)
		assert.NoError(t, err, "unmarshal message: %s", string(b))

		assert.Equal(t, m, m2, "messages are not equal")
	}
}
