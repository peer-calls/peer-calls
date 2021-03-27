package transport

import "github.com/peer-calls/peer-calls/server/identifiers"

type TrackJSON struct {
	ID       string             `json:"id"`
	StreamID string             `json:"streamID"`
	UserID   identifiers.UserID `json:"userId"`
	Codec    Codec              `json:"codec"`
}
