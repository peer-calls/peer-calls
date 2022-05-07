package message

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

var ErrUnknownMessageType = errors.New("unknown message type")

type JSON struct {
	Type Type `json:"type"`
	// Room this message is related to
	Room identifiers.RoomID `json:"room"`
	// Payload content
	Payload json.RawMessage `json:"payload"`
}

func (m Message) MarshalJSON() ([]byte, error) {
	var (
		payload []byte
		err     error
	)

	switch m.Type {
	case TypeHangUp:
		payload, err = json.Marshal(m.Payload.HangUp)
		err = errors.Trace(err)
	case TypeReady:
		payload, err = json.Marshal(m.Payload.Ready)
		err = errors.Trace(err)
	case TypeSignal:
		payload, err = json.Marshal(m.Payload.Signal)
		err = errors.Trace(err)
	case TypePing:
		payload, err = json.Marshal(m.Payload.Ping)
		err = errors.Trace(err)
	case TypePong:
		payload, err = json.Marshal(m.Payload.Pong)
		err = errors.Trace(err)
	case TypePubTrack:
		payload, err = json.Marshal(m.Payload.PubTrack)
		err = errors.Trace(err)
	case TypeSubTrack:
		payload, err = json.Marshal(m.Payload.SubTrack)
		err = errors.Trace(err)
	case TypeRoomJoin:
		payload, err = json.Marshal(m.Payload.RoomJoin)
		err = errors.Trace(err)
	case TypeRoomLeave:
		payload, err = json.Marshal(m.Payload.RoomLeave)
		err = errors.Trace(err)
	case TypeUsers:
		payload, err = json.Marshal(m.Payload.Users)
		err = errors.Trace(err)
	default:
		err = errors.Annotatef(ErrUnknownMessageType, "message: %+v", m)
	}

	if err != nil {
		return nil, errors.Trace(err)
	}

	j := JSON{
		Type:    m.Type,
		Room:    m.Room,
		Payload: json.RawMessage(payload),
	}

	b, err := json.Marshal(j)

	return b, errors.Annotatef(err, "message: %+v", m)
}

func (m *Message) UnmarshalJSON(b []byte) error {
	var j JSON

	err := json.Unmarshal(b, &j)
	if err != nil {
		return errors.Trace(err)
	}

	m.Room = j.Room
	m.Type = j.Type

	switch m.Type {
	case TypeHangUp:
		m.Payload.HangUp = &HangUp{}
		err = json.Unmarshal(j.Payload, m.Payload.HangUp)
		err = errors.Trace(err)
	case TypeReady:
		m.Payload.Ready = &Ready{}
		err = json.Unmarshal(j.Payload, m.Payload.Ready)
		err = errors.Trace(err)
	case TypeSignal:
		m.Payload.Signal = &UserSignal{}
		err = json.Unmarshal(j.Payload, m.Payload.Signal)
		err = errors.Trace(err)
	case TypePing:
		m.Payload.Ping = &Ping{}
	case TypePong:
		m.Payload.Pong = &Pong{}
	case TypePubTrack:
		m.Payload.PubTrack = &PubTrack{}
		err = json.Unmarshal(j.Payload, m.Payload.PubTrack)
		err = errors.Trace(err)
	case TypeSubTrack:
		m.Payload.SubTrack = &SubTrack{}
		err = json.Unmarshal(j.Payload, m.Payload.SubTrack)
		err = errors.Trace(err)
	case TypeRoomJoin:
		m.Payload.RoomJoin = &RoomJoin{}
		err = json.Unmarshal(j.Payload, m.Payload.RoomJoin)
		err = errors.Trace(err)
	case TypeRoomLeave:
		err = json.Unmarshal(j.Payload, &m.Payload.RoomLeave)
		err = errors.Trace(err)
	case TypeUsers:
		m.Payload.Users = &Users{}
		err = json.Unmarshal(j.Payload, m.Payload.Users)
		err = errors.Trace(err)
	default:
		err = errors.Trace(ErrUnknownMessageType)
	}

	return errors.Annotatef(err, "payload: %s", j.Payload)
}
