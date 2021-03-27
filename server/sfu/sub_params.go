package sfu

import (
	"github.com/peer-calls/peer-calls/server/identifiers"
)

type SubParams struct {
	// Room to which to subscribe to.
	Room        string
	PubClientID string
	TrackID     identifiers.TrackID
	SubClientID string
}
