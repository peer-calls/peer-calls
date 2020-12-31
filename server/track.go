package server

import (
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
)

type (
	Track          = transport.Track
	TrackInfo      = transport.TrackInfo
	TrackEventType = transport.TrackEventType
	TrackEvent     = transport.TrackEvent
)

type (
	SimpleTrack = transport.SimpleTrack
	UserTrack   = sfu.UserTrack
)

type UserIdentifiable interface {
	UserID() string
}
