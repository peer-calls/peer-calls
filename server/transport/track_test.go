package transport_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/stretchr/testify/assert"
)

func TestTrack(t *testing.T) {
	t1 := transport.NewSimpleTrack(3, 123, "a", "b")

	b, err := json.Marshal(t1)
	fmt.Println(string(b))
	assert.NoError(t, err)

	t2 := transport.SimpleTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t1, t2)

	assert.Equal(t, uint8(3), t2.PayloadType())
	assert.Equal(t, uint32(123), t2.SSRC())
	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.Label())
}
