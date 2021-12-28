package transport

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

type SimpleTrack struct {
	trackID identifiers.TrackID
	peerID  identifiers.PeerID
	codec   Codec
}

var _ Track = SimpleTrack{}

func NewSimpleTrack(id string, streamID string, codec Codec, peerID identifiers.PeerID) SimpleTrack {
	return SimpleTrack{
		trackID: identifiers.TrackID{
			ID:       id,
			StreamID: streamID,
		},
		peerID: peerID,
		codec:  codec,
	}
}

func (s SimpleTrack) SimpleTrack() SimpleTrack {
	return s
}

func (s SimpleTrack) PeerID() identifiers.PeerID {
	return s.peerID
}

func (s SimpleTrack) TrackID() identifiers.TrackID {
	return s.trackID
}

func (s SimpleTrack) Codec() Codec {
	return s.codec
}

func (s SimpleTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(TrackJSON{
		TrackID: s.trackID,
		PeerID:  s.peerID,
		Codec:   s.codec,
	})
}

func (s *SimpleTrack) UnmarshalJSON(data []byte) error {
	j := TrackJSON{}

	err := json.Unmarshal(data, &j)

	s.trackID = j.TrackID
	s.codec = j.Codec
	s.peerID = j.PeerID

	return errors.Annotatef(err, "unmarshal simple track json")
}
