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
	RemoveTrack(trackID TrackID) error
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

type trackCommon interface {
	Track() Track
	WriteRTCP([]rtcp.Packet) error
	ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
}

type TrackLocal interface {
	trackCommon
	WriteRTP(*rtp.Packet) (int, error)
}

type TrackRemote interface {
	trackCommon
	ReadRTP() (*rtp.Packet, interceptor.Attributes, error)
}

// // TODO implement Publisher.
// type Publisher interface {
// 	// AddTrack(TrackRemote) error
// 	// RemoveTrack(TrackID) error
// 	Tracks() []Track
// 	Subscribe(TrackID, Subscriber) error
// 	Unsubscribe(TrackID, Subscriber) error
// }

// // TODO implement Subscriber.
// type Subscriber interface {
// 	AddTrack(TrackID) (TrackLocal, error)
// 	RemoveTrack(TrackID) error
// }

type Transport2 interface {
	// RemoteTracksChannel contains remote tracks as they are received.
	RemoteTracksChannel() <-chan TrackRemote
	// AddTrack adds a new local track for writing.
	AddTrack(track Track) (TrackLocal, error)
	// RemoveTrack removes the local track.
	RemoveTrack(trackID TrackID) error
}

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error
}
