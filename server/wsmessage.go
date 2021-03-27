package server

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
)

type Serializer interface {
	Serialize(message Message) ([]byte, error)
}

type Deserializer interface {
	Deserialize([]byte) (Message, error)
}

// Simple message is a container for web-socket messages.
type Message struct {
	// Types 0-10 are reserved for base functionality, others can be used for
	// custom implementations.
	Type MessageType `json:"type"`
	// Room this message is related to
	Room identifiers.RoomID `json:"room"`
	// Payload content
	Payload interface{} `json:"payload"`
}

type MessageType string

const (
	MessageTypeHangUp   MessageType = "hangUp"
	MessageTypeReady    MessageType = "ready"
	MessageTypeSignal   MessageType = "signal"
	MessageTypePing     MessageType = "ping"
	MessageTypePubTrack MessageType = "pubTrack"
	MessageTypeSubTrack MessageType = "subTrack"

	MessageTypeRoomJoin  MessageType = "ws_room_join"
	MessageTypeRoomLeave MessageType = "ws_room_leave"

	MessageTypeUsers MessageType = "users"
)

func NewMessage(typ MessageType, room identifiers.RoomID, payload interface{}) Message {
	return Message{Type: typ, Room: room, Payload: payload}
}

func NewMessageRoomJoin(room identifiers.RoomID, clientID identifiers.ClientID, metadata string) Message {
	// FIXME strong types.
	return NewMessage(MessageTypeRoomJoin, room, map[string]string{
		"clientID": clientID.String(),
		"metadata": metadata,
	})
}

func NewMessageRoomLeave(room identifiers.RoomID, clientID identifiers.ClientID) Message {
	return NewMessage(MessageTypeRoomLeave, room, clientID)
}

type ByteSerializer struct{}

func (s ByteSerializer) Serialize(m Message) ([]byte, error) {
	b, err := json.Marshal(m)
	return b, errors.Annotate(err, "serialize")
}

func (s ByteSerializer) Deserialize(data []byte) (msg Message, err error) {
	err = json.Unmarshal(data, &msg)
	return msg, errors.Annotate(err, "deserialize")
}
