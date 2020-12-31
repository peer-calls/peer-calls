package transport

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
	TrackEventTypeSub
	TrackEventTypeUnsub
)

type TrackEvent struct {
	ClientID  string
	TrackInfo TrackInfo
	Type      TrackEventType
}

type TrackSub struct {
	SSRC uint32
}
