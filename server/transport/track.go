package transport

import (
	"github.com/peer-calls/peer-calls/server/identifiers"
)

type Track interface {
	TrackID() identifiers.TrackID
	PeerID() identifiers.PeerID
	Codec() Codec
	SimpleTrack() SimpleTrack
}
