package transport

import (
	"encoding/json"
	"fmt"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
)

type SimpleTrack struct {
	id       string
	streamID string
	mimeType string
	peerID   identifiers.PeerID

	uniqueID identifiers.TrackID

	codec Codec
}

var _ Track = SimpleTrack{}

func NewSimpleTrack(id string, streamID string, codec Codec, peerID identifiers.PeerID) SimpleTrack {
	return SimpleTrack{
		id:       id,
		streamID: streamID,
		peerID:   peerID,
		uniqueID: identifiers.TrackID(fmt.Sprintf("%s:%s", streamID, id)),
		codec:    codec,
	}
}

func (s SimpleTrack) SimpleTrack() SimpleTrack {
	return s
}

func (s SimpleTrack) ID() string {
	return s.id
}

func (s SimpleTrack) StreamID() string {
	return s.streamID
}

func (s SimpleTrack) PeerID() identifiers.PeerID {
	return s.peerID
}

func (s SimpleTrack) MimeType() string {
	return s.mimeType
}

func (s SimpleTrack) UniqueID() identifiers.TrackID {
	return s.uniqueID
}

func (s SimpleTrack) Codec() Codec {
	return s.codec
}

func (s SimpleTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(TrackJSON{
		ID:       s.id,
		StreamID: s.streamID,
		PeerID:   s.peerID,
		Codec:    s.codec,
	})
}

func (s *SimpleTrack) UnmarshalJSON(data []byte) error {
	j := TrackJSON{}

	err := json.Unmarshal(data, &j)

	s.id = j.ID
	s.streamID = j.StreamID
	s.codec = j.Codec
	s.peerID = j.PeerID
	s.uniqueID = identifiers.TrackID(fmt.Sprintf("%s:%s", j.StreamID, j.ID))

	return errors.Annotatef(err, "unmarshal simple track json")
}
