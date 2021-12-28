package sfu

import (
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

type SubParams struct {
	// Room to which to subscribe to.
	Room        identifiers.RoomID
	PubClientID identifiers.ClientID
	TrackID     identifiers.TrackID
	SubClientID identifiers.ClientID
}
