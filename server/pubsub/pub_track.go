package pubsub

type PubTrack struct {
	ClientID string `json:"clientId"`
	UserID   string `json:"userId"`
	SSRC     uint32 `json:"ssrc"`
}

func newPubTrack(pb *pub) PubTrack {
	var userID string

	userTrack, ok := pb.track.(userIdentifiable)
	if ok {
		userID = userTrack.UserID()
	}

	return PubTrack{
		SSRC:     pb.track.SSRC(),
		ClientID: pb.clientID,
		UserID:   userID,
	}
}
