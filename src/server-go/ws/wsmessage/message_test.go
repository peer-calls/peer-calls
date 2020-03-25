package wsmessage_test

import (
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/ws/wsmessage"
	"github.com/stretchr/testify/assert"
)

func TestMessageSerializeDeserialize(t *testing.T) {
	typ := ^uint16(0)
	payload := []byte{1, 2, 3}
	m1 := wsmessage.NewMessage(typ, payload)
	assert.Equal(t, typ, m1.Type())
	assert.Equal(t, payload, m1.Payload())
	var s wsmessage.ByteSerializer
	serialized := s.Serialize(m1)
	m2 := s.Deserialize(serialized)
	assert.Equal(t, typ, m2.Type())
	assert.Equal(t, payload, m2.Payload())
}

func TestNewMessageRoomJoin(t *testing.T) {
	room := "test"
	m1 := wsmessage.NewMessageRoomJoin(room)
	assert.Equal(t, wsmessage.MessageTypeRoomJoin, m1.Type())
	assert.Equal(t, []byte(room), m1.Payload())
}

func TestNewMessageRoomLeave(t *testing.T) {
	room := "test"
	m1 := wsmessage.NewMessageRoomLeave(room)
	assert.Equal(t, wsmessage.MessageTypeRoomLeave, m1.Type())
	assert.Equal(t, []byte(room), m1.Payload())
}
