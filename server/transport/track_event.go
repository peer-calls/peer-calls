package transport

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
	TrackEventTypeSub
	TrackEventTypeUnsub
)
