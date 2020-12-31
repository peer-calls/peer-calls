package pubsub

type PubTrack struct {
	ClientID string `json:"clientId"`
	UserID   string `json:"userId"`
	SSRC     uint32 `json:"ssrc"`
}
