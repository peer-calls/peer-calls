package server

import (
	"encoding/json"
)

const (
	MessageTypeRoomJoin  string = "ws_room_join"
	MessageTypeRoomLeave string = "ws_room_leave"
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
	Type string `json:"type"`
	// Room this message is related to
	Room string `json:"room"`
	// Payload content
	Payload interface{} `json:"payload"`
}

func NewMessage(typ string, room string, payload interface{}) Message {
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

const uint64Size = uint64(8)

func (s ByteSerializer) Serialize(m Message) ([]byte, error) {
	return json.Marshal(m)
}

func (s ByteSerializer) Deserialize(data []byte) (msg Message, err error) {
	err = json.Unmarshal(data, &msg)
	return
}
