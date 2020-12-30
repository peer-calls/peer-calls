package transport

import "github.com/pion/webrtc/v3"

type Track interface {
	PayloadType() uint8
	SSRC() uint32
	ID() string
	Label() string
}

type TrackInfo struct {
	Track Track
	Kind  webrtc.RTPCodecType
	Mid   string
}

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
)

type TrackEvent struct {
	TrackInfo TrackInfo
	Type      TrackEventType
}
