package sfu

import "github.com/peer-calls/peer-calls/server/transport"

type SubParams struct {
	// Room to which to subscribe to.
	Room        string
	PubClientID string
	TrackID     transport.TrackID
	SubClientID string
}
