package identifiers

import (
	"sort"
	"strings"
)

type TrackID struct {
	// ID corresponds to track label.
	ID string `json:"id"`
	// Stream corresponds to stream ID.
	StreamID string `json:"streamId"`
}

type RoomID string

// ClientID is the remote client ID that's connected to the server.
type ClientID string

// PeerID is the ID of the remote peer that published the track. In other
// words it's the ID of the origin of this track.
type PeerID string

type ClientIDs []ClientID

func (r RoomID) String() string {
	return string(r)
}

func (c ClientID) String() string {
	return string(c)
}

var _ sort.Interface = ClientIDs(nil)

func (c ClientIDs) Len() int {
	return len(c)
}

func (c ClientIDs) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c ClientIDs) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

const ServerNodePrefix = "node:"

func (c ClientID) IsServer() bool {
	return strings.HasPrefix(string(c), ServerNodePrefix)
}
