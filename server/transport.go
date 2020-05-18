package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type Transport interface {
	WriteRTCP([]rtcp.Packet) error
	WriteRTP(*rtp.Packet) (int, error)
	// TrackEventsChannel() <-chan TrackEvent
	RTPChannel() <-chan *rtp.Packet
	RTCPChannel() <-chan rtcp.Packet
	// MessagesChannel() <-chan webrtc.DataChannelMessage
}
