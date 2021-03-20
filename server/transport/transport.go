package transport

import (
	"github.com/pion/interceptor"
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

	Subscriber
	Publisher

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

// type MediaTransport interface {
// 	AddTrack(track Track) (TrackLocal, error)
// 	RemoveTrack(trackID TrackID) error
// 	WriteRTCP([]rtcp.Packet) error
// 	WriteRTP(*rtp.Packet) (int, error)
// 	RTPChannel() <-chan *rtp.Packet
// 	RTCPChannel() <-chan []rtcp.Packet
// }

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
}

// // TODO implement Publisher.
// type Publisher interface {
// 	// AddTrack(TrackRemote) error
// 	// RemoveTrack(TrackID) error
// 	Tracks() []Track
// 	Subscribe(TrackID, Subscriber) error
// 	Unsubscribe(TrackID, Subscriber) error
// }

type Publisher interface {
	// RemoteTracks() []TrackInfo
	RemoteTracksChannel() <-chan TrackRemote
}

// // TODO implement Subscriber.
type Subscriber interface {
	LocalTracks() []TrackWithMID
	AddTrack(Track) (TrackLocal, error)
	RemoveTrack(TrackID) error
}

// type Transport2 interface {
// 	// RemoteTracksChannel contains remote tracks as they are received.
// 	RemoteTracksChannel() <-chan TrackRemote
// 	// AddTrack adds a new local track for writing.
// 	AddTrack(track Track) (TrackLocal, error)
// 	// RemoveTrack removes the local track.
// 	RemoveTrack(trackID TrackID) error
// }

type DataTransport interface {
	MessagesChannel() <-chan webrtc.DataChannelMessage
	Send(message webrtc.DataChannelMessage) <-chan error
}
