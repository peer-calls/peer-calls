package identifiers

import (
	"sort"
	"strings"
)

type TrackID string

type RoomID string

type ClientID string

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
