package server

import (
	"testing"

	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/stretchr/testify/assert"
)

func TestMessageSerializeDeserialize(t *testing.T) {
	typ := MessageType("test-type")
	room := identifiers.RoomID("test-room")
	payload := "test-payload"
	m1 := NewMessage(typ, room, payload)
	assert.Equal(t, typ, m1.Type)
	assert.Equal(t, payload, m1.Payload)
	assert.Equal(t, room, m1.Room)
	var s ByteSerializer
	serialized, err := s.Serialize(m1)
	assert.Nil(t, err)
	m2, err := s.Deserialize(serialized)
	assert.Nil(t, err)
	assert.Equal(t, typ, m2.Type)
	assert.Equal(t, payload, m2.Payload)
	assert.Equal(t, room, m2.Room)
}

func TestNewMessageRoomJoin(t *testing.T) {
	room := identifiers.RoomID("test")
	clientID := identifiers.ClientID("client1")
	metadata := "mydata"
	m1 := NewMessageRoomJoin(room, clientID, metadata)
	assert.Equal(t, MessageTypeRoomJoin, m1.Type)
	assert.Equal(t, room, m1.Room)
	payload, ok := m1.Payload.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, clientID.String(), payload["clientID"])
	assert.Equal(t, metadata, payload["metadata"])
}

func TestNewMessageRoomLeave(t *testing.T) {
	room := identifiers.RoomID("test")
	clientID := identifiers.ClientID("client1")
	m1 := NewMessageRoomLeave(room, clientID)
	assert.Equal(t, MessageTypeRoomLeave, m1.Type)
	assert.Equal(t, room, m1.Room)
	assert.Equal(t, clientID, m1.Payload)
}
