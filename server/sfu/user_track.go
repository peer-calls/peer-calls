package sfu

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
)

type UserTrack struct {
	payloadType uint8
	ssrc        uint32
	id          string
	label       string
	userID      string
	roomID      string
}

func NewUserTrack(track transport.Track, userID string, roomID string) UserTrack {
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

var _ transport.Track = UserTrack{}
