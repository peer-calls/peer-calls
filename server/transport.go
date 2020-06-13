package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type Transport interface {
	ClientID() string
	MediaTransport
	DataTransport
	MetadataTransport
	Closable
}

type MetadataTransport interface {
	TrackEventsChannel() <-chan TrackEvent
	LocalTracks() []TrackInfo
	RemoteTracks() []TrackInfo
	AddTrack(payloadType uint8, ssrc uint32, id string, label string) error
	RemoveTrack(ssrc uint32) error
}

type Closable interface {
	Close() error
	CloseChannel() <-chan struct{}
}

type MediaTransport interface {
	WriteRTCP([]rtcp.Packet) error
	WriteRTP(*rtp.Packet) (int, error)
	RTPChannel() <-chan *rtp.Packet
	RTCPChannel() <-chan rtcp.Packet
}

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message []byte) error
	SendText(message string) error
}
