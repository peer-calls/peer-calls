package transport

import (
	"github.com/pion/interceptor"
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

	DataTransport

	RemoteTracksChannel() <-chan TrackRemote

	LocalTracks() []TrackWithMID
	AddTrack(Track) (TrackLocal, Sender, error)
	RemoveTrack(TrackID) error

	RTCPWriter

	Closable
}

type Closable interface {
	Close() error
	Done() <-chan struct{}
}

type trackCommon interface {
	Track() Track
}

type TrackLocal interface {
	trackCommon
	Write([]byte) (int, error)
	WriteRTP(*rtp.Packet) error
}

type TrackRemote interface {
	trackCommon
	ReadRTP() (*rtp.Packet, interceptor.Attributes, error)
	SSRC() webrtc.SSRC
	RID() string
}

type Sender interface {
	ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
}

type RTCPWriter interface {
	WriteRTCP([]rtcp.Packet) error
}

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error
}
