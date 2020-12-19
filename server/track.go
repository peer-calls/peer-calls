package server

import (
	"encoding/json"

	"github.com/juju/errors"
)

type Track interface {
	PayloadType() uint8
	SSRC() uint32
	ID() string
	Label() string
}

type UserIdentifiable interface {
	UserID() string
}

type RoomIdentifiable interface {
	RoomID() string
}

type UserRoomIdentifiable interface {
	UserIdentifiable
	RoomIdentifiable
}

type SimpleTrack struct {
	payloadType uint8
	ssrc        uint32
	id          string
	label       string
}

type TrackJSON struct {
	PayloadType uint8  `json:"payloadType"`
	SSRC        uint32 `json:"ssrc"`
	ID          string `json:"id"`
	Label       string `json:"label"`
}

var _ Track = SimpleTrack{}

func NewSimpleTrack(payloadType uint8, ssrc uint32, id string, label string) SimpleTrack {
	return SimpleTrack{
		payloadType: payloadType,
		ssrc:        ssrc,
		id:          id,
		label:       label,
	}
}

func (s SimpleTrack) PayloadType() uint8 {
	return s.payloadType
}

func (s SimpleTrack) SSRC() uint32 {
	return s.ssrc
}

func (s SimpleTrack) ID() string {
	return s.id
}

func (s SimpleTrack) Label() string {
	return s.label
}

func (s SimpleTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(TrackJSON{
		PayloadType: s.payloadType,
		SSRC:        s.ssrc,
		ID:          s.id,
		Label:       s.label,
	})
}

func (s *SimpleTrack) UnmarshalJSON(data []byte) error {
	j := TrackJSON{}

	err := json.Unmarshal(data, &j)

	s.payloadType = j.PayloadType
	s.ssrc = j.SSRC
	s.id = j.ID
	s.label = j.Label

	return errors.Annotatef(err, "unmarshal simple track json")
}

type UserTrack struct {
	payloadType uint8
	ssrc        uint32
	id          string
	label       string
	userID      string
	roomID      string
}

func NewUserTrack(track Track, userID string, roomID string) UserTrack {
	return UserTrack{
		payloadType: track.PayloadType(),
		ssrc:        track.SSRC(),
		id:          track.ID(),
		label:       track.Label(),
		userID:      userID,
		roomID:      roomID,
	}
}

type UserTrackJSON struct {
	PayloadType uint8  `json:"payloadType"`
	SSRC        uint32 `json:"ssrc"`
	ID          string `json:"id"`
	Label       string `json:"label"`
	UserID      string `json:"userId"`
	RoomID      string `json:"roomId"`
}

func (t UserTrack) PayloadType() uint8 {
	return t.payloadType
}

func (t UserTrack) SSRC() uint32 {
	return t.ssrc
}

func (t UserTrack) ID() string {
	return t.id
}

func (t UserTrack) Label() string {
	return t.label
}

func (t UserTrack) UserID() string {
	return t.userID
}

func (t UserTrack) RoomID() string {
	return t.roomID
}

func (t UserTrack) MarshalJSON() ([]byte, error) {
	j := UserTrackJSON{
		PayloadType: t.payloadType,
		SSRC:        t.ssrc,
		ID:          t.id,
		Label:       t.label,
		UserID:      t.userID,
		RoomID:      t.roomID,
	}

	return json.Marshal(j)
}

func (t *UserTrack) UnmarshalJSON(data []byte) error {
	j := UserTrackJSON{}

	err := json.Unmarshal(data, &j)

	t.payloadType = j.PayloadType
	t.ssrc = j.SSRC
	t.id = j.ID
	t.label = j.Label
	t.userID = j.UserID
	t.roomID = j.RoomID

	return errors.Annotatef(err, "unmarshal user track json")
}

var _ Track = UserTrack{}
var _ UserRoomIdentifiable = UserTrack{}
