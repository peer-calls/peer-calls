package transport

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Type int

const (
	TypeWebRTC Type = iota + 1
	TypeServer
)

type Transport interface {
	ClientID() string
	Type() Type
	MediaTransport
	DataTransport
	MetadataTransport
	Closable
}

type MetadataTransport interface {
	TrackEventsChannel() <-chan TrackEvent
	LocalTracks() []TrackInfo
	RemoteTracks() []TrackInfo
	AddTrack(track Track) error
	RemoveTrack(ssrc uint32) error
}

type Closable interface {
	Close() error
	Done() <-chan struct{}
}

type MediaTransport interface {
	WriteRTCP([]rtcp.Packet) error
	WriteRTP(*rtp.Packet) (int, error)
	RTPChannel() <-chan *rtp.Packet
	RTCPChannel() <-chan []rtcp.Packet
}

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error
}
