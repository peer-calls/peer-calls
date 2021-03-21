package servertransport

import (
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type trackEvent struct {
	ClientID string                   `json:"clientID"`
	Track    transport.SimpleTrack    `json:"track"`
	Type     transport.TrackEventType `json:"type"`
	SSRC     webrtc.SSRC              `json:"ssrc"`
}
