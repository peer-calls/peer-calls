package wsmessage

import "encoding/binary"

const (
	MessageTypeRoomJoin uint16 = iota + 1
	MessageTypeRoomLeave
)

type Message interface {
	Type() uint16
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
	typ uint16
	// Payload content
	message []byte
}

func NewMessage(typ uint16, message []byte) SimpleMessage {
	return SimpleMessage{typ, message}
}

func NewMessageRoomJoin(clientID string) SimpleMessage {
	return NewMessage(MessageTypeRoomJoin, []byte(clientID))
}

func NewMessageRoomLeave(clientID string) SimpleMessage {
	return NewMessage(MessageTypeRoomLeave, []byte(clientID))
}

func (s SimpleMessage) Type() uint16 {
	return s.typ
}

func (s SimpleMessage) Payload() []byte {
	return s.message
}

type ByteSerializer struct{}

func (s ByteSerializer) Serialize(m Message) []byte {
	typeSize := 2
	message := m.Payload()
	data := make([]byte, typeSize+len(message))
	binary.LittleEndian.PutUint16(data[0:typeSize], m.Type())
	copy(data[typeSize:], message)
	return data
}

func (s ByteSerializer) Deserialize(data []byte) Message {
	typeSize := 2
	var m SimpleMessage
	m.typ = binary.LittleEndian.Uint16(data[0:typeSize])
	m.message = data[typeSize:]
	return m
}
