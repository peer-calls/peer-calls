package transport

import (
	"encoding/json"
	"testing"

	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/stretchr/testify/assert"
)

func TestSimpleTrack(t *testing.T) {
	codec := Codec{
		MimeType:    "audio/opus",
		ClockRate:   48000,
		Channels:    2,
		SDPFmtpLine: "a=b",
	}

	t1 := NewSimpleTrack("a", "b", codec, "user-1")
	assert.Equal(t, identifiers.TrackID("b:a"), t1.UniqueID())

	b, err := json.Marshal(t1)
	assert.NoError(t, err)

	t2 := SimpleTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)

	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.StreamID())
	assert.Equal(t, identifiers.TrackID("b:a"), t2.UniqueID())
	assert.Equal(t, identifiers.PeerID("user-1"), t2.PeerID())
	assert.Equal(t, codec, t2.Codec())
}
