package transport

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/pion/webrtc/v3"
)

type Track interface {
	PayloadType() uint8
	SSRC() uint32
	ID() string
	Label() string
}

type TrackInfo struct {
	Track Track
	Kind  webrtc.RTPCodecType
	Mid   string
}

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
)

type TrackEvent struct {
	TrackInfo TrackInfo
	Type      TrackEventType
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
