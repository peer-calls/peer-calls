package transport

type TrackJSON struct {
	ID       string `json:"id"`
	StreamID string `json:"streamID"`
	UserID   string `json:"userId"`
	MimeType string `json:"mimeType"`
}
