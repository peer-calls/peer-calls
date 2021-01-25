package pubsub

import (
	"fmt"

	"github.com/peer-calls/peer-calls/server/transport"
)

type clientTrack struct {
	// ClientID of the publisher.
	ClientID string

	// TrackID is the unique ID of a published track.
	TrackID transport.TrackID
}

func (p clientTrack) String() string {
	return fmt.Sprintf("%s:%s", p.ClientID, p.TrackID)
}
