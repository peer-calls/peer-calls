package transport

type Track interface {
	UniqueID() TrackID
	ID() string
	StreamID() string
	MimeType() string
	UserID() string
}

type TrackID string
