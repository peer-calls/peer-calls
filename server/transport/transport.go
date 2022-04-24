package transport

import (
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
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
	ClientID() identifiers.ClientID
	Type() Type

	DataTransport

	// RemoteTracksChannel might never be closed.
	// Use Done() in select when reading from this channel to prevent deadlocks.
	RemoteTracksChannel() <-chan TrackRemoteWithRTCPReader

	LocalTracks() []TrackWithMID

	AddTrack(Track) (TrackLocal, RTCPReader, error)
	RemoveTrack(identifiers.TrackID) error

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

type RTCPReader interface {
	ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
}

type TrackRemoteWithRTCPReader struct {
	TrackRemote TrackRemote
	RTCPReader  RTCPReader
}

type RTCPWriter interface {
	WriteRTCP([]rtcp.Packet) error
}

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error
}
