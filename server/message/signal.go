package message

import (
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
)

type Signal struct {
	Candidate          *webrtc.ICECandidateInit `json:"candidate,omitempty"`
	Renegotiate        bool                     `json:"renegotiate,omitempty"`
	Type               SignalType               `json:"type"`
	SDP                string                   `json:"sdp,omitempty"`
	TransceiverRequest *TransceiverRequest      `json:"transceiverRequest,omitempty"`
}

type SignalType string

const (
	SignalTypeCandidate          SignalType = "candidate"
	SignalTypeTransceiverRequest SignalType = "transceiverRequest"
	SignalTypeRenegotiate        SignalType = "renegotiate"
	SignalTypeOffer              SignalType = "offer"
	SignalTypePranswer           SignalType = "pranswer"
	SignalTypeAnswer             SignalType = "answer"
	SignalTypeRollback           SignalType = "rollback"
)

func NewSignalTypeFromSDPType(sdpType webrtc.SDPType) (SignalType, bool) {
	switch sdpType {
	case webrtc.SDPTypeOffer:
		return SignalTypeOffer, true
	case webrtc.SDPTypePranswer:
		return SignalTypePranswer, true
	case webrtc.SDPTypeAnswer:
		return SignalTypeAnswer, true
	case webrtc.SDPTypeRollback:
		return SignalTypeRollback, true
	default:
		return "", false
	}
}

func (s SignalType) SDPType() (webrtc.SDPType, bool) {
	switch s {
	case SignalTypeOffer:
		return webrtc.SDPTypeOffer, true
	case SignalTypePranswer:
		return webrtc.SDPTypePranswer, true
	case SignalTypeAnswer:
		return webrtc.SDPTypeAnswer, true
	case SignalTypeRollback:
		return webrtc.SDPTypeRollback, true
	default:
		return 0, false
	}
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
