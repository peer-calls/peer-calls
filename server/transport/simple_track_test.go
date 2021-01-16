package transport

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleTrack(t *testing.T) {
	t1 := NewSimpleTrack("xyz", 3, 123, "a", "b")

	b, err := json.Marshal(t1)
	assert.NoError(t, err)

	t2 := SimpleTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)

	assert.Equal(t, "xyz", t2.UserID())
	assert.Equal(t, uint8(3), t2.PayloadType())
	assert.Equal(t, uint32(123), t2.SSRC())
	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.Label())
}
