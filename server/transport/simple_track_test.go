package transport

import (
	"encoding/json"
	"testing"

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
	assert.Equal(t, TrackID("b:a"), t1.UniqueID())

	b, err := json.Marshal(t1)
	assert.NoError(t, err)

	t2 := SimpleTrack{}
	err = json.Unmarshal(b, &t2)
	assert.NoError(t, err)

	assert.Equal(t, "a", t2.ID())
	assert.Equal(t, "b", t2.StreamID())
	assert.Equal(t, TrackID("b:a"), t2.UniqueID())
	assert.Equal(t, "user-1", t2.UserID())
	assert.Equal(t, codec, t2.Codec())
}
