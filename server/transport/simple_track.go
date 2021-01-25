package transport

import (
	"encoding/json"
	"fmt"

	"github.com/juju/errors"
)

type SimpleTrack struct {
	id       string
	streamID string
	mimeType string
	userID   string

	uniqueID TrackID
}

var _ Track = SimpleTrack{}

func NewSimpleTrack(id string, streamID string, mimeType string, userID string) SimpleTrack {
	return SimpleTrack{
		id:       id,
		streamID: streamID,
		mimeType: mimeType,
		userID:   userID,
		uniqueID: TrackID(fmt.Sprintf("%s:%s", streamID, id)),
	}
}

func (s SimpleTrack) ID() string {
	return s.id
}

func (s SimpleTrack) StreamID() string {
	return s.streamID
}

func (s SimpleTrack) UserID() string {
	return s.userID
}

func (s SimpleTrack) MimeType() string {
	return s.mimeType
}

func (s SimpleTrack) UniqueID() TrackID {
	return s.uniqueID
}

func (s SimpleTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(TrackJSON{
		ID:       s.id,
		StreamID: s.streamID,
		MimeType: s.mimeType,
		UserID:   s.userID,
	})
}

func (s *SimpleTrack) UnmarshalJSON(data []byte) error {
	j := TrackJSON{}

	err := json.Unmarshal(data, &j)

	s.id = j.ID
	s.streamID = j.StreamID
	s.mimeType = j.MimeType
	s.userID = j.UserID

	return errors.Annotatef(err, "unmarshal simple track json")
}
