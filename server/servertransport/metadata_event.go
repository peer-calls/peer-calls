package servertransport

import (
	"fmt"
)

type metadataEvent struct {
	// Type must always be set
	Type metadataEventType `json:"type"`

	// Track will be set only when Type is metadataEventTypeTrackEvent.
	TrackEvent trackEvent `json:"trackEvent"`
}

type metadataEventType int

const (
	// Track event contains the information about tracks.
	metadataEventTypeTrack metadataEventType = iota + 1
)

func (m metadataEventType) String() string {
	switch m {
	case metadataEventTypeTrack:
		return "TrackEvent"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// type initEventJSON struct {
// 	ClientID string
// }

// type byeEventJSON struct{}
