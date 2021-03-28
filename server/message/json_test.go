package message_test

import (
	"encoding/json"
	"testing"

	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/message"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func TestMessage_JSON(t *testing.T) {
	messages := []message.Message{
		{
			Type: message.TypeHangUp,
			Room: "test",
			Payload: message.Payload{
				HangUp: &message.HangUp{},
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
				Signal: &message.Signal{
					Candidate: message.Candidate{
						Candidate:     "a",
						SDPMlineIndex: "b",
						SDPMid:        "c",
					},
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.Signal{
					TransceiverRequest: &message.TransceiverRequest{
						Kind:      message.TransceiverRequestKindAudio,
						Direction: webrtc.RTPTransceiverDirectionSendrecv,
					},
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.Signal{
					Renegotiate: true,
				},
			},
		},
		{
			Type: message.TypeSignal,
			Room: "test",
			Payload: message.Payload{
				Signal: &message.Signal{
					Type: message.SignalTypeOffer,
					SDP:  "-sdp-",
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
					TrackID:     identifiers.TrackID("123"),
					PubClientID: identifiers.ClientID("client123"),
					UserID:      identifiers.UserID("user123"),
					Type:        transport.TrackEventTypeAdd,
				},
			},
		},
		{
			Type: message.TypeSubTrack,
			Room: "test",
			Payload: message.Payload{
				SubTrack: &message.SubTrack{
					TrackID:     identifiers.TrackID("123"),
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
					ClientID: identifiers.UserID("user123"),
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
				RoomLeave: "user123",
			},
		},
	}

	for _, m := range messages {
		b, err := json.Marshal(m)
		assert.NoError(t, err, "marshal message: %+v", m)

		var m2 message.Message

		err = json.Unmarshal(b, &m2)
		assert.NoError(t, err, "unmarshal message: %s", string(b))
	}
}
