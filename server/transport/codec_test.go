package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodec(t *testing.T) {
	audio := Codec{
		MimeType:    "audio/opus",
		ClockRate:   48000,
		Channels:    2,
		SDPFmtpLine: "",
	}

	video := Codec{
		MimeType:    "video/vp8",
		ClockRate:   90000,
		Channels:    0,
		SDPFmtpLine: "",
	}

	assert.Equal(t, TrackKindAudio, audio.TrackKind())
	assert.Equal(t, TrackKindVideo, video.TrackKind())
}
