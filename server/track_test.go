package server

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrack_SimpleTrack(t *testing.T) {
	t1 := NewSimpleTrack(3, 123, "a", "b")

	b, err := json.Marshal(t1)
	fmt.Println(string(b))
	assert.NoError(t, err)

	t2 := SimpleTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)

	assert.Equal(t, uint8(3), t2.PayloadType())
	assert.Equal(t, uint32(123), t2.SSRC())
	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.Label())

	// Can be unmarshaled to UserTrack
	t3 := UserTrack{}
	err = json.Unmarshal(b, &t3)

	assert.Equal(t, uint8(3), t3.PayloadType())
	assert.Equal(t, uint32(123), t3.SSRC())
	assert.Equal(t, "a", t3.ID())
	assert.Equal(t, "b", t3.Label())
}

func TestTrack_UserTrack(t *testing.T) {
	track := NewSimpleTrack(3, 123, "a", "b")
	t1 := NewUserTrack(track, "c", "d")

	b, err := json.Marshal(t1)
	assert.NoError(t, err)

	t2 := UserTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)

	assert.Equal(t, uint8(3), t2.PayloadType())
	assert.Equal(t, uint32(123), t2.SSRC())
	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.Label())
	assert.Equal(t, "c", t2.UserID())
	assert.Equal(t, "d", t2.RoomID())

	// Can be unmarshaled to SimpleTrack
	t3 := SimpleTrack{}
	err = json.Unmarshal(b, &t3)

	assert.Equal(t, uint8(3), t3.PayloadType())
	assert.Equal(t, uint32(123), t3.SSRC())
	assert.Equal(t, "a", t3.ID())
	assert.Equal(t, "b", t3.Label())
}
