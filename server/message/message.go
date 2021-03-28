package message

import (
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/transport"
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

type Props struct {
	// Types 0-10 are reserved for base functionality, others can be used for
	// custom implementations.
	Type Type `json:"type"`
	// Room this message is related to
	Room identifiers.RoomID `json:"room"`
	// Payload content. Depending on the Type, only one field will be set.
	Payload Payload `json:"payload"`
}

// Payload should only have a single field set, depending on the type of the
// message.
type Payload struct {
	HangUp *HangUp
	Ready  *Ready
	Signal *Signal
	Ping   *Ping

	PubTrack *PubTrack
	SubTrack *SubTrack

	RoomJoin  *RoomJoin
	RoomLeave identifiers.ClientID

	Users *Users
}

type RoomJoin struct {
	ClientID identifiers.UserID `json:"userId"`
	Metadata string             `json:"metadata"`
}

type Type string

const (
	TypeHangUp Type = "hangUp"
	TypeReady  Type = "ready"
	TypeSignal Type = "signal"
	TypePing   Type = "ping"

	TypePubTrack Type = "pubTrack"
	TypeSubTrack Type = "subTrack"

	TypeRoomJoin  Type = "ws_room_join"
	TypeRoomLeave Type = "ws_room_leave"

	TypeUsers Type = "users"
)

type HangUp struct{}

type Ready struct {
	Nickname string `json:"nickname"`
}

type Ping struct{}

type Users struct {
	Initiator identifiers.ClientID            `json:"initiator"`
	PeerIDs   []identifiers.ClientID          `json:"peerIDs"`
	Nicknames map[identifiers.ClientID]string `json:"nicknames"`
}

type PubTrack struct {
	TrackID     identifiers.TrackID  `json:"trackId"`
	PubClientID identifiers.ClientID `json:"pubClientId"`
	UserID      identifiers.UserID   `json:"userId"`
	// Type can contain only Add or Remove.
	Type transport.TrackEventType `json:"type"`
}

type SubTrack struct {
	TrackID     identifiers.TrackID  `json:"trackId"`
	PubClientID identifiers.ClientID `json:"pubClientId"`
	// Type can contain only Sub or Unsub.
	Type transport.TrackEventType `json:"type"`
}
