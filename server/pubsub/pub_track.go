package pubsub

import (
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/transport"
)

// PubTrack will be emitted as an event after a track is published or
// unpublished.
type PubTrack struct {
	// ClientID is the remote client ID that we got the track from. In most
	// cases, when a track was received from another WebRTCTransport connected to
	// the same server, it will be the same as PeerID. If the track was received
	// from another server (using servertransport), it will be the ID of the
	// remote server.
	ClientID identifiers.ClientID `json:"clientId"`
	// PeerID is the ID of the remote peer that published the track. In other
	// words it's the ID of the origin of this track.
	PeerID identifiers.PeerID `json:"peerId"`
	// TrackID contains unique track identifier, consisting of track ID and
	// StreamID.
	TrackID identifiers.TrackID `json:"trackId"`
	// Kind is the track kind (audio or video).
	Kind transport.TrackKind `json:"kind"`
}

// newPubTrack creates a new instance of PubTrack.
func newPubTrack(pubClientID identifiers.ClientID, track transport.Track) PubTrack {
	return PubTrack{
		ClientID: pubClientID,
		PeerID:   track.PeerID(),
		TrackID:  track.TrackID(),
		Kind:     track.Codec().TrackKind(),
	}
}
