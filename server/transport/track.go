package transport

type Track interface {
	UniqueID() TrackID
	ID() string
	StreamID() string
	UserID() string
	Codec() Codec
	SimpleTrack() SimpleTrack
}

type Codec struct {
	MimeType    string `json:"mimeType"`
	ClockRate   uint32 `json:"clockRate"`
	Channels    uint16 `json:"channels"`
	SDPFmtpLine string `json:"sdpFmtpLine"`
}

type TrackID string
