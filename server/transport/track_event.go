package transport

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
	TrackEventTypeSub
	TrackEventTypeUnsub
)

type TrackEvent struct {
	ClientID     string
	TrackWithMID TrackWithMID
	Type         TrackEventType
}
