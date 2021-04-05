package transport

import (
	"strings"

	"github.com/pion/webrtc/v3"
)

type Codec struct {
	MimeType    string `json:"mimeType"`
	ClockRate   uint32 `json:"clockRate"`
	Channels    uint16 `json:"channels"`
	SDPFmtpLine string `json:"sdpFmtpLine"`
}

func (c Codec) TrackKind() TrackKind {
	if strings.HasPrefix(c.MimeType, "audio/") {
		return TrackKindAudio
	}

	return TrackKindVideo
}

type TrackKind string

const (
	TrackKindAudio TrackKind = "audio"
	TrackKindVideo TrackKind = "video"
)

func NewTrackKind(codecType webrtc.RTPCodecType) TrackKind {
	if codecType == webrtc.RTPCodecTypeAudio {
		return TrackKindAudio
	}

	return TrackKindVideo
}

func (t TrackKind) RTPCodecType() webrtc.RTPCodecType {
	if t == TrackKindAudio {
		return webrtc.RTPCodecTypeAudio
	}

	return webrtc.RTPCodecTypeVideo
}
