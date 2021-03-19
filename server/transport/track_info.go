package transport

type TrackWithMID struct {
	Track
	// Kind  webrtc.RTPCodecType
	mid string
}

func NewTrackWithMID(track Track, mid string) TrackWithMID {
	return TrackWithMID{track, mid}
}

func (t TrackWithMID) MID() string {
	return t.mid
}
