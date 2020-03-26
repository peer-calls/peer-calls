package wsmessage_test

import (
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/stretchr/testify/assert"
)

func TestMessageSerializeDeserialize(t *testing.T) {
	typ := "test-type"
	room := "test-room"
	payload := []byte{1, 2, 3}
	m1 := wsmessage.NewMessage(typ, room, payload)
	assert.Equal(t, typ, m1.Type())
	assert.Equal(t, payload, m1.Payload())
	assert.Equal(t, room, m1.Room())
	var s wsmessage.ByteSerializer
	serialized, err := s.Serialize(m1)
	assert.Nil(t, err)
	m2, err := s.Deserialize(serialized)
	assert.Nil(t, err)
	assert.Equal(t, typ, m2.Type())
	assert.Equal(t, payload, m2.Payload())
	assert.Equal(t, room, m2.Room())
}

func TestNewMessageRoomJoin(t *testing.T) {
	room := "test"
	clientID := "client1"
	m1 := wsmessage.NewMessageRoomJoin(room, clientID)
	assert.Equal(t, wsmessage.MessageTypeRoomJoin, m1.Type())
	assert.Equal(t, room, m1.Room())
	assert.Equal(t, []byte(clientID), m1.Payload())
}

func TestNewMessageRoomLeave(t *testing.T) {
	room := "test"
	clientID := "client1"
	m1 := wsmessage.NewMessageRoomLeave(room, clientID)
	assert.Equal(t, wsmessage.MessageTypeRoomLeave, m1.Type())
	assert.Equal(t, room, m1.Room())
	assert.Equal(t, []byte(clientID), m1.Payload())
}
