package pubsub

import (
	"github.com/peer-calls/peer-calls/v4/server/transport"
)

type PubTrackEvent struct {
	PubTrack PubTrack                 `json:"pubTrack"`
	Type     transport.TrackEventType `json:"type"`
}
