package server

import (
	"fmt"

	"github.com/pion/webrtc/v2"
)

type TransceiverRequestPayload struct {
	TransceiverRequest struct {
		Kind webrtc.RTPCodecType        `json:"kind"`
		Init *webrtc.RtpTransceiverInit `json:"init"`
	} `json:"transceiverRequest"`
}

type TransceiverRequestJSON struct {
	TransceiverRequest struct {
		Kind string `json:"kind"`
		Init struct {
			Direction string `json:"direction,omitempty"`
		} `json:"init"`
	} `json:"transceiverRequest"`
}

type Renegotiate struct {
	Renegotiate bool `json:"renegotiate"`
}

type Candidate struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type Payload struct {
	UserID string      `json:"userId"`
	Signal interface{} `json:"signal"`
}

func NewPayloadSDP(userID string, sessionDescription webrtc.SessionDescription) Payload {
	return Payload{
		UserID: userID,
		Signal: sessionDescription,
	}
}

func NewPayloadRenegotiate(userID string) Payload {
	return Payload{
		UserID: userID,
		Signal: Renegotiate{
			Renegotiate: true,
		},
	}
}

func NewTransceiverRequest(userID string, kind webrtc.RTPCodecType, direction webrtc.RTPTransceiverDirection) Payload {
	signal := TransceiverRequestJSON{}

	signal.TransceiverRequest.Kind = kind.String()
	signal.TransceiverRequest.Init.Direction = direction.String()

	return Payload{
		UserID: userID,
		Signal: signal,
	}
}

func newCandidate(candidate interface{}) (c Candidate, err error) {
	candidateMap, ok := candidate.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("Expected candidate to be a map: %#v", candidate)
		return
	}

	candidateValue, ok := candidateMap["candidate"]
	if !ok {
		err = fmt.Errorf("Expected candidate.candidate %#v", candidate)
	}

	candidateString, ok := candidateValue.(string)
	if !ok {
		err = fmt.Errorf("Expected candidate.candidate to be a string: %#v", candidate)
		return
	}
	sdpMLineIndexValue, ok := candidateMap["sdpMLineIndex"]
	if !ok {
		err = fmt.Errorf("Expected candidate.sdpMLineIndex to exist: %#v", sdpMLineIndexValue)
		return
	}
	sdpMLineIndex, ok := sdpMLineIndexValue.(float64)
	if !ok {
		err = fmt.Errorf("Expected candidate.sdpMLineIndex be float64: %T", sdpMLineIndexValue)
		return
	}

	sdpMid, ok := candidateMap["sdpMid"].(string)
	var sdpMidPtr *string
	if ok {
		sdpMidPtr = &sdpMid
	}

	sdpMLineIndexUint16 := uint16(sdpMLineIndex)
	c.Candidate.Candidate = candidateString
	c.Candidate.SDPMLineIndex = &sdpMLineIndexUint16
	c.Candidate.SDPMid = sdpMidPtr

	return
}

func newTransceiverRequest(transceiverRequest interface{}) (r TransceiverRequestPayload, err error) {
	transceiverRequestMap, ok := transceiverRequest.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("Transceiver request is not a map: %#v", transceiverRequest)
		return
	}

	kind, ok := transceiverRequestMap["kind"]
	if !ok {
		err = fmt.Errorf("Transceiver request kind not found: %#v", transceiverRequest)
		return
	}
	kindString, ok := kind.(string)
	if !ok {
		err = fmt.Errorf("Transceiver request kind should be a string: %#v", transceiverRequest)
		return
	}

	r.TransceiverRequest.Kind = webrtc.RTPCodecTypeVideo
	if kindString == "audio" {
		r.TransceiverRequest.Kind = webrtc.RTPCodecTypeAudio
	}

	if init, ok := transceiverRequestMap["init"]; ok {
		initMap, ok := init.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("Expectd init to be a map: %#v", transceiverRequest)
		}

		var transceiverInit webrtc.RtpTransceiverInit
		for key, value := range initMap {
			switch key {
			case "direction":
				switch value {
				case "sendrecv":
					transceiverInit.Direction = webrtc.RTPTransceiverDirectionSendrecv
				case "sendonly":
					transceiverInit.Direction = webrtc.RTPTransceiverDirectionSendonly
				case "recvonly":
					transceiverInit.Direction = webrtc.RTPTransceiverDirectionRecvonly
				case "inactive":
					transceiverInit.Direction = webrtc.RTPTransceiverDirectionInactive
				}
			}
		}

		r.TransceiverRequest.Init = &transceiverInit
	}

	return
}

func newRenegotiate() Renegotiate {
	return Renegotiate{
		Renegotiate: true,
	}
}

func newSDP(sdpType interface{}, signal map[string]interface{}) (s webrtc.SessionDescription, err error) {
	sdpTypeString, ok := sdpType.(string)
	if !ok {
		err = fmt.Errorf("Expected signal.type to be string: %#v", signal)
		return
	}

	sdp, ok := signal["sdp"]
	if !ok {
		err = fmt.Errorf("Expected signal.sdp: %#v", signal)
	}

	sdpString, ok := sdp.(string)
	if !ok {
		err = fmt.Errorf("Expected signal.sdp to be string: %#v", signal)
		return
	}
	s.SDP = sdpString

	switch sdpTypeString {
	case "offer":
		s.Type = webrtc.SDPTypeOffer
	case "answer":
		s.Type = webrtc.SDPTypeAnswer
	case "pranswer":
		err = fmt.Errorf("Handling of pranswer signal implemented")
	case "rollback":
		err = fmt.Errorf("Handling of rollback signal not implemented")
	default:
		err = fmt.Errorf("Unknown sdp type: %s", sdpString)
	}

	return
}

func NewPayloadFromMap(payload map[string]interface{}) (p Payload, err error) {
	userID, ok := payload["userId"].(string)
	if !ok {
		err = fmt.Errorf("No userId property in payload: %#v", payload)
		return
	}
	signal, ok := payload["signal"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("No signal property in payload: %#v", payload)
		return
	}

	var value interface{}

	if candidate, ok := signal["candidate"]; ok {
		value, err = newCandidate(candidate)
	} else if _, ok := signal["renegotiate"]; ok {
		value = newRenegotiate()
	} else if transceiverRequest, ok := signal["transceiverRequest"]; ok {
		value, err = newTransceiverRequest(transceiverRequest)
	} else if sdpType, ok := signal["type"]; ok {
		value, err = newSDP(sdpType, signal)
	} else {
		err = fmt.Errorf("Unexpected signal message: %#v", payload)
		return
	}

	if err != nil {
		return
	}

	p.UserID = userID
	p.Signal = value
	return
}
