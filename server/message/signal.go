package message

import "github.com/pion/webrtc/v3"

type Signal struct {
	Candidate          Candidate           `json:"candidate,omitempty"`
	Renegotiate        bool                `json:"renegotiate,omitempty"`
	Type               SignalType          `json:"type,omitempty"`
	SDP                string              `json:"string"`
	TransceiverRequest *TransceiverRequest `json:"transceiverRequest,omitempty"`
}

type Candidate struct {
	Candidate     string `json:"candidate"`
	SDPMlineIndex string `json:"sdpMLineIndex"`
	SDPMid        string `json:"sdpMid"`
}

type Renegotiate struct {
	Renegotiate bool `json:"renegotiate"`
}

type TransceiverRequest struct {
	Kind      TransceiverRequestKind         `json:"kind"`
	Direction webrtc.RTPTransceiverDirection `json:"direction"`
}

type TransceiverRequestKind string

const (
	TransceiverRequestKindAudio = "audio"
	TransceiverRequestKindVideo = "video"
)

type SignalType string

const (
	SignalTypeOffer    SignalType = "offer"
	SignalTypeAnswer   SignalType = "answer"
	SignalTypePranswer SignalType = "pranswer"
	SignalTypeRollback SignalType = "rollback"
)
