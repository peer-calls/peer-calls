package server

type Track interface {
	PayloadType() uint8
	SSRC() uint32
	ID() string
	Label() string
}

type SimpleTrack struct {
	payloadType uint8
	ssrc        uint32
	id          string
	label       string
}

var _ Track = SimpleTrack{}

func NewSimpleTrack(payloadType uint8, ssrc uint32, id string, label string) SimpleTrack {
	return SimpleTrack{payloadType, ssrc, id, label}
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
