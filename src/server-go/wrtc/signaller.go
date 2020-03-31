package wrtc

import (
	"fmt"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/negotiator"
	"github.com/pion/webrtc/v2"
)

type SignalCandidate struct {
	UserID string    `json:"userId"`
	Signal Candidate `json:"signal"`
}

type SignalSDP struct {
	UserID string                    `json:"userId"`
	Signal webrtc.SessionDescription `json:"signal"`
}

type SignalRenegotiate struct {
	UserID string      `json:"userId"`
	Signal Renegotiate `json:"signal"`
}

type SignalTransceiverRequest struct {
	UserID string             `json:"userId"`
	Signal TransceiverRequest `json:"signal"`
}

type TransceiverRequest struct {
	TransceiverRequest struct {
		Kind string                    `json:"kind"`
		Init webrtc.RtpTransceiverInit `json:"init"`
	} `json:"transceiverRequest"`
}

type Renegotiate struct {
	Renegotiate bool `json:"renegotiate"`
}

type Candidate struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type PeerConnection interface {
	OnICECandidate(func(*webrtc.ICECandidate))
	OnSignalingStateChange(func(webrtc.SignalingState))
	AddICECandidate(webrtc.ICECandidateInit) error
	AddTransceiverFromKind(codecType webrtc.RTPCodecType, init ...webrtc.RtpTransceiverInit) (*webrtc.RTPTransceiver, error)
	SetRemoteDescription(webrtc.SessionDescription) error
	SetLocalDescription(webrtc.SessionDescription) error
	CreateOffer(*webrtc.OfferOptions) (webrtc.SessionDescription, error)
	CreateAnswer(*webrtc.AnswerOptions) (webrtc.SessionDescription, error)
}

type Signaller struct {
	peerConnection PeerConnection
	localPeerID    string
	onSignal       func(signal interface{})
	negotiator     *negotiator.Negotiator
}

var log = logger.GetLogger("wrtc")

func NewSignaller(
	initiator bool,
	peerConnection PeerConnection,
	localPeerID string,
	onSignal func(signal interface{}),
) *Signaller {
	s := Signaller{
		peerConnection: peerConnection,
		localPeerID:    localPeerID,
		onSignal:       onSignal,
	}

	negotiator := negotiator.NewNegotiator(
		initiator,
		peerConnection,
		s.handleLocalOffer,
		s.handleLocalRequestNegotiation,
	)

	s.negotiator = negotiator

	if initiator {
		s.negotiator.Negotiate()
	}
	// peerConnection.OnICECandidate(s.handleICECandidate)

	return &s
}

func (s *Signaller) handleICECandidate(c *webrtc.ICECandidate) {
	if c == nil {
		return
	}

	payload := SignalCandidate{
		UserID: s.localPeerID,
		Signal: Candidate{
			Candidate: c.ToJSON(),
		},
	}

	log.Printf("Got ice candidate from server peer: %s", payload)
	s.onSignal(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) (err error) {
	signal, _ := payload["signal"].(map[string]interface{})
	remotePeerID, _ := payload["userId"].(string)

	if remotePeerID != s.localPeerID {
		return fmt.Errorf("Peer2Server only sends signals to server as peer")
	}

	if candidate, ok := signal["candidate"]; ok {
		log.Printf("Got remote ice candidate")
		err = s.handleSignalCandidate(remotePeerID, candidate)
	} else if _, ok := signal["renegotiate"]; ok {
		log.Printf("Got renegotiation request")
		s.Negotiate()
	} else if transceiverRequest, ok := signal["transceiverRequest"]; ok {
		err = s.handleTransceiverRequest(transceiverRequest)
	} else if sdpType, ok := signal["type"]; ok {
		log.Printf("Got remote signal (type: %s)", sdpType)
		err = s.handleSDP(sdpType, signal["sdp"])
	} else {
		err = fmt.Errorf("Unexpected signal message: %#v", payload)
	}

	return
}

func (s *Signaller) handleTransceiverRequest(transceiverRequest interface{}) (err error) {
	transceiverRequestMap, ok := transceiverRequest.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Invalid transceiver request type:  %#v", transceiverRequest)
	}
	kind, ok := transceiverRequestMap["kind"]
	if !ok {
		return fmt.Errorf("No kind field for transceiver request: %#v", transceiverRequest)
	}
	kindString, ok := kind.(string)
	if !ok {
		return fmt.Errorf("Invalid kind field type for transceiver request: %s", kind)
	}
	// TODO ignoring direction and sendencodings
	switch kindString {
	case "video":
		log.Printf("Got transceiver request (type: video)")
		_, err = s.peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeVideo,
			webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
		)
		if err != nil {
			return fmt.Errorf("Error adding video transceiver: %s", err)
		}
		s.Negotiate()
	case "audio":
		log.Printf("Got transceiver request (type: audio)")
		_, err = s.peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeAudio,
			webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
		)
		if err != nil {
			return fmt.Errorf("Error adding audio transceiver: %s", err)
		}
		s.Negotiate()
	default:
		return fmt.Errorf("invalid transceiver kind: %s", kindString)
	}
	return nil
}

func (s *Signaller) handleSignalCandidate(targetClientID string, candidate interface{}) (err error) {
	// log.Printf("Got client ice candidate: %#v", candidate)
	candidateMap, ok := candidate.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Expected ice candidate to be a map")
	}

	candidateString, _ := candidateMap["candidate"].(string)
	sdpMLineIndex, _ := candidateMap["sdpMLineIndex"].(uint16)
	sdpMid, _ := candidateMap["sdpMid"].(string)

	iceCandidate := webrtc.ICECandidateInit{
		Candidate:     candidateString,
		SDPMLineIndex: &sdpMLineIndex,
		SDPMid:        &sdpMid,
	}

	// log.Printf("Parsed ice candidate: %#v", iceCandidate)

	err = s.peerConnection.AddICECandidate(iceCandidate)
	return
}

func (s *Signaller) handleSDP(sdpType interface{}, sdp interface{}) (err error) {
	sdpTypeString, _ := sdpType.(string)
	sdpString, _ := sdp.(string)
	sessionDescription := webrtc.SessionDescription{}
	sessionDescription.SDP = sdpString

	switch sdpTypeString {
	case "offer":
		sessionDescription.Type = webrtc.SDPTypeOffer
		return s.handleOffer(sessionDescription)
		// mediaEngine.PopulateFromSDP(sdp) TODO figure out if we need this
		// videoCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
		// audioCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)
	case "answer":
		sessionDescription.Type = webrtc.SDPTypeAnswer
		return s.handleAnswer(sessionDescription)
	case "pranswer":
		return fmt.Errorf("Handling of pranswer signal implemented")
	case "rollback":
		return fmt.Errorf("Handling of rollback signal not implemented")
	default:
		return fmt.Errorf("Unknown sdp type: %s", sdpString)
	}

}

func (s *Signaller) handleOffer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("Error setting remote description: %w", err)
	}
	answer, err := s.peerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("Error creating answer: %w", err)
	}
	if err := s.peerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("Error setting local description: %w", err)
	}

	answerSignalSDP := SignalSDP{
		UserID: s.localPeerID,
		Signal: answer,
	}
	// log.Printf("Sending answer: %#v", answerSignalSDP)
	s.onSignal(answerSignalSDP)
	return nil
}

func (s *Signaller) handleLocalRequestNegotiation() {
	log.Println("Sending renegotiation request to initiator")
	s.onSignal(SignalRenegotiate{
		UserID: s.localPeerID,
		Signal: Renegotiate{
			Renegotiate: true,
		},
	})
}

func (s *Signaller) handleLocalOffer(offer webrtc.SessionDescription, err error) {
	log.Println("Created local offer")
	if err != nil {
		log.Printf("Error creating local offer: %s", err)
		// TODO abort connection
		return
	}

	err = s.peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Printf("Error setting local description from local offer: %s", err)
		// TODO abort connection
		return
	}

	offerSignalSDP := SignalSDP{
		UserID: s.localPeerID,
		Signal: offer,
	}
	s.onSignal(offerSignalSDP)
}

// TODO check offer voice activation detection feature of webrtc

// Create an offer and send it to remote peer
func (s *Signaller) Negotiate() {
	s.negotiator.Negotiate()
}

func (s *Signaller) handleAnswer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("Error setting remote description: %w", err)
	}
	return nil
}
