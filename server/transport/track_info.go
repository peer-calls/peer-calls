package transport

import "github.com/pion/webrtc/v3"

type TrackInfo struct {
	Track Track
	Kind  webrtc.RTPCodecType
	Mid   string
}
