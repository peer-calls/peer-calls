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

	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error

	RemoteTracksChannel() <-chan TrackRemote

	LocalTracks() []TrackWithMID
	AddTrack(Track) (TrackLocal, Sender, error)
	RemoveTrack(TrackID) error

	WriteRTCP([]rtcp.Packet) error

	Closable
}

// type MetadataTransport interface {
// 	// TrackEventsChannel() <-chan TrackEvent
// 	// AddTrack(track Track) (TrackLocal, error)
// 	// RemoveTrack(trackID TrackID) error
// }

type Closable interface {
	Close() error
	Done() <-chan struct{}
}

type trackCommon interface {
	Track() Track
	// WriteRTCP([]rtcp.Packet) error
	// ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
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
