package pubsub

import "github.com/peer-calls/peer-calls/server/transport"

type PubTrack struct {
	ClientID string            `json:"clientId"`
	UserID   string            `json:"userId"`
	TrackID  transport.TrackID `json:"trackId"`
}

func newPubTrack(pb *pub) PubTrack {
	return PubTrack{
		ClientID: pb.clientID,
		TrackID:  pb.track.UniqueID(),
		UserID:   pb.track.UserID(),
	}
}
