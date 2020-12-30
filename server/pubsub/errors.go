package pubsub

import "errors"

var (
	ErrTrackNotFound       = errors.New("track not found")
	ErrSubNotFound         = errors.New("subscriber not found")
	ErrSubscribeToOwnTrack = errors.New("cannot subscribe to own track")
)
