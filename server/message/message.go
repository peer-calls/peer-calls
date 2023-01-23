package message

import (
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/transport"
)

type Message struct {
	// Types 0-10 are reserved for base functionality, others can be used for
	// custom implementations.
	Type Type
	// Room this message is related to
	Room identifiers.RoomID
	// Payload content
	Payload Payload
}

func NewPing(roomID identifiers.RoomID) Message {
	return Message{
		Type: TypePing,
		Room: roomID,
		Payload: Payload{
			Ping: &Ping{},
		},
	}
}

func NewReady(roomID identifiers.RoomID, payload Ready) Message {
	return Message{
		Type: TypeReady,
		Room: roomID,
		Payload: Payload{
			Ready: &payload,
		},
	}
}

func NewHangUp(roomID identifiers.RoomID, payload HangUp) Message {
	return Message{
		Type: TypeHangUp,
		Room: roomID,
		Payload: Payload{
			HangUp: &payload,
		},
	}
}

func NewRoomJoin(roomID identifiers.RoomID, payload RoomJoin) Message {
	return Message{
		Type: TypeRoomJoin,
		Room: roomID,
		Payload: Payload{
			RoomJoin: &payload,
		},
	}
}

func NewRoomLeave(roomID identifiers.RoomID, clientID identifiers.ClientID) Message {
	return Message{
		Type: TypeRoomLeave,
		Room: roomID,
		Payload: Payload{
			RoomLeave: clientID,
		},
	}
}

func NewUsers(roomID identifiers.RoomID, payload Users) Message {
	return Message{
		Type: TypeUsers,
		Room: roomID,
		Payload: Payload{
			Users: &payload,
		},
	}
}

func NewPubTrack(roomID identifiers.RoomID, payload PubTrack) Message {
	return Message{
		Type: TypePubTrack,
		Room: roomID,
		Payload: Payload{
			PubTrack: &payload,
		},
	}
}

func NewSubTrack(roomID identifiers.RoomID, payload SubTrack) Message {
	return Message{
		Type: TypeSubTrack,
		Room: roomID,
		Payload: Payload{
			SubTrack: &payload,
		},
	}
}

func NewSignal(roomID identifiers.RoomID, payload UserSignal) Message {
	return Message{
		Type: TypeSignal,
		Room: roomID,
		Payload: Payload{
			Signal: &payload,
		},
	}
}

type UserSignal struct {
	PeerID identifiers.ClientID `json:"peerId"`
	Signal Signal               `json:"signal"`
}

// Payload should only have a single field set, depending on the type of the
// message.
type Payload struct {
	HangUp *HangUp
	// Ready is sent from the client to the server.
	Ready  *Ready
	Signal *UserSignal
	Ping   *Ping
	Pong   *Pong

	PubTrack *PubTrack
	SubTrack *SubTrack

	// RoomJoin is only sent to other server-side clients in the same room.
	RoomJoin *RoomJoin
	// RoomLeave is only sent to other server-side clients in the same room.
	RoomLeave identifiers.ClientID

	// Users is sent as a response to Ready.
	// TODO use PubTrack instead.
	Users *Users
}

type RoomJoin struct {
	ClientID identifiers.ClientID `json:"peerId"`
	Metadata string               `json:"metadata"`
}

type Type string

const (
	TypeHangUp Type = "hangUp"
	TypeReady  Type = "ready"
	TypeSignal Type = "signal"
	TypePing   Type = "ping"
	TypePong   Type = "pong"

	TypePubTrack Type = "pubTrack"
	TypeSubTrack Type = "subTrack"

	TypeRoomJoin  Type = "wsRoomJoin"
	TypeRoomLeave Type = "wsRoomLeave"

	TypeUsers Type = "users"
)

type HangUp struct {
	PeerID identifiers.ClientID `json:"peerId"`
}

type Ready struct {
	Nickname string `json:"nickname"`
}

type Ping struct{}

type Pong struct{}

// The only thing that's not easy to handle this way are nicknames.
type Users struct {
	Initiator identifiers.ClientID            `json:"initiator"`
	PeerIDs   []identifiers.ClientID          `json:"peerIds"`
	Nicknames map[identifiers.ClientID]string `json:"nicknames"`
}

// PubTrack will be sent to the clients whenever a track is published or
// unpublished.
//
// Note about PubClientID, PeerID and SourceID: these values will be the same
// for Mesh, but different for SFU.
//
// In the case of a single-node SFU:
// - PubClientID and PeerID will be the same and define the original track
//   source.
//
// In the case of multi-node SFU:
// - PubClientID will be ID of the transport (WebRTC or Server) that published
//   the track to current node.
// - PeerID will be the ID of the original track source (most likely a WebRTC
//   transport)
type PubTrack struct {
	// TrackID is the unique track identifier.
	TrackID identifiers.TrackID `json:"trackId"`
	// PubClientID is the ID of the remote client that published the track.
	PubClientID identifiers.ClientID `json:"pubClientId"`
	// PeerID is the original track source.
	PeerID identifiers.PeerID `json:"peerId"`
	// Kind defines whether this is an audio or video track.
	Kind transport.TrackKind `json:"kind"`
	// Type can contain only Add or Remove.
	Type transport.TrackEventType `json:"type"`
}

type SubTrack struct {
	TrackID     identifiers.TrackID  `json:"trackId"`
	PubClientID identifiers.ClientID `json:"pubClientId"`
	// Type can contain only Sub or Unsub.
	Type transport.TrackEventType `json:"type"`
}
