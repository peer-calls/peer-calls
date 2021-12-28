package servertransport

import (
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
)

type trackEvent struct {
	ClientID identifiers.ClientID     `json:"clientID"`
	Track    transport.SimpleTrack    `json:"track"`
	Type     transport.TrackEventType `json:"type"`
	SSRC     webrtc.SSRC              `json:"ssrc"`

	// TODO RTCPHeaderextensions
	// TODO RTCPFeedback
	// TODO PayloadType??

	// Payload type is determined when TrackLocal.Bind is called.
	//
	// Question: Do we need to implement something like this for server to server
	// transport?
	//
	// If we have something like:
	//
	// +---------+ Track  +-----+ Track  +------------+ Track  +-------------+ Track  +-----+ Track  +---------+
	// | Browser | -----> | PC1 | -----> | Transport 1| -----> | Transport 2 | -----> | PC2 | -----> | Browser |
	// +---------+        +-----+        +------------+        +-------------+        +-----+        +---------+
	//
	// Browser and PC1 need to negotiate.
	// Browser and PC2 need to negotiate.
	//
	// Example: we only support OPUS and VP8.
	//
	// When adding tracks from PC1 to PC2, the transport doens't really need to
	// know anything about this.
	//
	// However, when a TrackLocal is added to PC2, PC2 will call Bind on it.
	//
	// Perhaps we could use "Bind" instead of the Subscribe?
	//
	// Design difference:
	//
	// We pass around transport.Track and create a new instance of TrackLocal
	// every time.
	//
	// We _could_ pass around the actual track instance like pion/webrtc does
	// in the examples.
	//
	// Then when a track was actually bound by the peer connection, the transport
	// 2 sends a metadata message to transport 1 and transport 1 starts sending.
	//
	// But should the transport 1 communicate anything to PC1?
	// I don't think so at this point.
}
