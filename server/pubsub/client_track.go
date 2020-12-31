package pubsub

import "fmt"

type clientTrack struct {
	// ClientID of the publisher.
	ClientID string

	// SSRC of the published track.
	SSRC uint32
}

func (p clientTrack) String() string {
	return fmt.Sprintf("%s:%d", p.ClientID, p.SSRC)
}
