package servertransport

import (
	"fmt"

	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type metadataEvent struct {
	// Type must always be set
	Type metadataEventType `json:"type"`

	// TrackEvent will be set only when Type is metadataEventTypeTrackEvent.
	TrackEvent *trackEventJSON `json:"trackEvent"`
}

type metadataEventType int

const (
	// TrackEvent contains the information about tracks.
	metadataEventTypeTrackEvent metadataEventType = iota + 1
	// GetTracks event will return all tracks.
	metadataEventTypeInit
)

func (m metadataEventType) String() string {
	switch m {
	case metadataEventTypeTrackEvent:
		return "TrackEvent"
	case metadataEventTypeInit:
		return "Init"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// trackEventJSON is used instead of TrackEvent because JSON cannot deserialize
// to Track interface, so a UserTrack is used.
type trackEventJSON struct {
	ClientID  string
	TrackInfo trackInfoJSON
	Type      transport.TrackEventType
}

func newTrackEventJSON(trackEvent transport.TrackEvent) trackEventJSON {
	// TODO watch out for possible panics.
	track := trackEvent.TrackInfo.Track.(sfu.UserTrack)

	return trackEventJSON{
		ClientID: trackEvent.ClientID,
		TrackInfo: trackInfoJSON{
			Track: track,
			Kind:  trackEvent.TrackInfo.Kind,
			Mid:   trackEvent.TrackInfo.Mid,
		},
		Type: trackEvent.Type,
	}
}

// trackEvent converts the trackEventJSON to TrackEvent.
func (t trackEventJSON) trackEvent(clientID string) transport.TrackEvent {
	return transport.TrackEvent{
		ClientID: clientID,
		TrackInfo: transport.TrackInfo{
			Track: t.TrackInfo.Track,
			Kind:  t.TrackInfo.Kind,
			Mid:   t.TrackInfo.Mid,
		},
		Type: t.Type,
	}
}

type trackInfoJSON struct {
	Track sfu.UserTrack
	Kind  webrtc.RTPCodecType
	Mid   string
}
