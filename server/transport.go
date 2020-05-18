package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type Transport interface {
	WriteRTCP([]rtcp.Packet) error
	WriteRTP(*rtp.Packet) (int, error)
	NewTrack(payloadType uint8, ssrc uint32, id string, label string) (*webrtc.Track, error)
	AddTrack(track *webrtc.Track) (<-chan rtcp.Packet, error)
	RemoveTrack(*webrtc.Track) error
	OnTrack(func(*webrtc.Track))
	Mid(ssrc uint32) string
}
