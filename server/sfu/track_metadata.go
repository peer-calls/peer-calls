package sfu

import "github.com/peer-calls/peer-calls/server/identifiers"

type TrackMetadata struct {
	Mid      string             `json:"mid"`
	UserID   identifiers.UserID `json:"userId"`
	StreamID string             `json:"streamId"`
	Kind     string             `json:"kind"`
}
