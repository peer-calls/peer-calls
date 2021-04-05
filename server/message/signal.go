package message

import (
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type Signal struct {
	Candidate          *webrtc.ICECandidateInit `json:"candidate,omitempty"`
	Renegotiate        bool                     `json:"renegotiate,omitempty"`
	Type               webrtc.SDPType           `json:"type,omitempty"`
	SDP                string                   `json:"sdp,omitempty"`
	TransceiverRequest *TransceiverRequest      `json:"transceiverRequest,omitempty"`
}

type TransceiverRequest struct {
	Kind transport.TrackKind `json:"kind"`
	Init TransceiverInit     `json:"init"`
}

type TransceiverInit struct {
	Direction Direction `json:"direction,omitempty"`
}

type Direction string

const (
	DirectionSendRecv Direction = "sendrecv"
	DirectionSendOnly Direction = "sendonly"
	DirectionRecvOnly Direction = "recvonly"
	DirectionInactive Direction = "inactive"
)

func (d Direction) RTPTransceiverDirection() (webrtc.RTPTransceiverDirection, bool) {
	switch d {
	case "sendrecv":
		return webrtc.RTPTransceiverDirectionSendrecv, true
	case "sendonly":
		return webrtc.RTPTransceiverDirectionSendonly, true
	case "recvonly":
		return webrtc.RTPTransceiverDirectionRecvonly, true
	case "inactive":
		return webrtc.RTPTransceiverDirectionInactive, true
	}

	return webrtc.RTPTransceiverDirection(0), false
}
