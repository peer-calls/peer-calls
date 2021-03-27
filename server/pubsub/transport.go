package pubsub

import (
	"github.com/peer-calls/peer-calls/server/transport"
)

// Transport only defines a subset of methods from transport.Transport to make
// mocking in testing easier.
type Transport interface {
	ClientID() string

	AddTrack(track transport.Track) (transport.TrackLocal, transport.RTCPReader, error)
	RemoveTrack(trackID transport.TrackID) error
}

// Assert that Transport is compatible with the transport.Transport.
var _ Transport = transport.Transport(nil)
