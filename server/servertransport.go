package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type ServerTransport struct {
}

func NewServerTransport() *ServerTransport {
	return &ServerTransport{}
}

func (s *ServerTransport) WriteRTCP(packets []rtcp.Packet) error {
	return NotImplementedErr
}

func (s *ServerTransport) WriteRTP(pkt *rtp.Packet) (int, error) {
	return 0, NotImplementedErr
}

func (s *ServerTransport) NewTrack(payloadType uint8, ssrc uint32, id string, label string) (*webrtc.Track, error) {
	return nil, NotImplementedErr
}

func (s *ServerTransport) AddTrack(track *webrtc.Track) (<-chan rtcp.Packet, error) {
	return nil, NotImplementedErr
}

func (s *ServerTransport) RemoveTrack(*webrtc.Track) error {
	return NotImplementedErr
}

func (s *ServerTransport) OnTrack(func(*webrtc.Track)) {
}

func (s *ServerTransport) Mid(ssrc uint32) string {
	return ""
}

var _ Transport = &ServerTransport{}
