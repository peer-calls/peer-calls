package server

import (
	"encoding/json"

	"github.com/juju/errors"
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
	Room string `json:"room"`
	// Payload content
	Payload interface{} `json:"payload"`
}

type MessageType string

const (
	MessageTypeHangUp MessageType = "hangUp"
	MessageTypeReady  MessageType = "ready"
	MessageTypeSignal MessageType = "signal"
	MessageTypePing   MessageType = "ping"

	MessageTypeRoomJoin  MessageType = "ws_room_join"
	MessageTypeRoomLeave MessageType = "ws_room_leave"

	MessageTypeUsers MessageType = "users"
)

func NewMessage(typ MessageType, room string, payload interface{}) Message {
	return Message{Type: typ, Room: room, Payload: payload}
}

func NewMessageRoomJoin(room string, clientID string, metadata string) Message {
	return NewMessage(MessageTypeRoomJoin, room, map[string]string{
		"clientID": clientID,
		"metadata": metadata,
	})
}

func NewMessageRoomLeave(room string, clientID string) Message {
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
