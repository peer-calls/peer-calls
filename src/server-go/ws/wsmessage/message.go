package wsmessage

import (
	"encoding/binary"
)

const (
	MessageTypeRoomJoin  string = "ws_room_join"
	MessageTypeRoomLeave string = "ws_room_leave"
)

type Message interface {
	Type() string
	Room() string
	Payload() []byte
}

type Serializer interface {
	Serialize(message Message) []byte
}

type Deserializer interface {
	Deserialize([]byte) Message
}

// Simple message is a container for web-socket messages.
type SimpleMessage struct {
	// Types 0-10 are reserved for base functionality, others can be used for
	// custom implementations.
	typ string
	// Room this message is related to
	room string
	// Payload content
	payload []byte
}

func NewMessage(typ string, room string, payload []byte) SimpleMessage {
	return SimpleMessage{typ, room, payload}
}

func NewMessageRoomJoin(room string, clientID string) SimpleMessage {
	return NewMessage(MessageTypeRoomJoin, room, []byte(clientID))
}

func NewMessageRoomLeave(room string, clientID string) SimpleMessage {
	return NewMessage(MessageTypeRoomLeave, room, []byte(clientID))
}

func (s SimpleMessage) Type() string {
	return s.typ
}

func (s SimpleMessage) Payload() []byte {
	return s.payload
}

func (s SimpleMessage) Room() string {
	return s.room
}

type ByteSerializer struct{}

const uint64Size = uint64(8)

func (s ByteSerializer) Serialize(m Message) []byte {
	room := []byte(m.Room())
	typ := []byte(m.Type())
	payload := m.Payload()

	totalSize := uint64Size*3 + uint64(len(room)) + uint64(len(typ)) + uint64(len(payload))
	data := make([]byte, totalSize)

	offset := uint64(0)
	writeBytes(data, room, &offset)
	writeBytes(data, typ, &offset)
	writeBytes(data, payload, &offset)

	return data
}

func (s ByteSerializer) Deserialize(data []byte) Message {
	var m SimpleMessage

	offset := uint64(0)

	m.room = string(readBytes(data, &offset))
	m.typ = string(readBytes(data, &offset))
	m.payload = readBytes(data, &offset)

	return m
}

func readBytes(data []byte, offset *uint64) (value []byte) {
	end := *offset + uint64Size
	if end > uint64(len(data)) {
		// TODO convert this to error
		return
	}
	length := binary.LittleEndian.Uint64(data[*offset:end])
	*offset += uint64Size
	end = *offset + length
	if end > uint64(len(data)) {
		// TODO convert this to error
		return
	}
	value = data[*offset : *offset+length]
	*offset += length
	return
}

func writeBytes(data []byte, value []byte, offset *uint64) {
	length := uint64(len(value))
	binary.LittleEndian.PutUint64(data[*offset:*offset+uint64Size], length)
	*offset += uint64Size
	copy(data[*offset:*offset+length], value)
	*offset += length
}
