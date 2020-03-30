package wrtc

import (
	"fmt"

	"github.com/jeremija/peer-calls/src/server-go/logger"
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

type Candidate struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type PeerConnection interface {
	OnICECandidate(func(*webrtc.ICECandidate))
	AddICECandidate(webrtc.ICECandidateInit) error
	SetRemoteDescription(webrtc.SessionDescription) error
	SetLocalDescription(webrtc.SessionDescription) error
	CreateOffer(*webrtc.OfferOptions) (webrtc.SessionDescription, error)
	CreateAnswer(*webrtc.AnswerOptions) (webrtc.SessionDescription, error)
}

type Signaller struct {
	peerConnection    PeerConnection
	localPeerID       string
	onSignalSDP       func(signal SignalSDP) error
	onSignalCandidate func(signal SignalCandidate)
}

var log = logger.GetLogger("wrtc")

func NewSignaller(
	peerConnection PeerConnection,
	localPeerID string,
	onSignalSDP func(signal SignalSDP) error,
	onSignalCandidate func(signal SignalCandidate),
) *Signaller {
	s := Signaller{
		peerConnection:    peerConnection,
		localPeerID:       localPeerID,
		onSignalSDP:       onSignalSDP,
		onSignalCandidate: onSignalCandidate,
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
	s.onSignalCandidate(payload)
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
	} else if sdpType, ok := signal["type"]; ok {
		log.Printf("Got remote signal (type: %s)", sdpType)
		err = s.handleSDP(sdpType, signal["sdp"])
	} else {
		err = fmt.Errorf("Unexpected signal message: %#v", payload)
	}

	return
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
	err = s.onSignalSDP(answerSignalSDP)
	return err
}

// TODO check offer voice activation detection feature of webrtc

// Create an offer and send it to remote peer
func (s *Signaller) Negotiate() (err error) {
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("Error creating offer: %w", err)
	}
	s.peerConnection.SetLocalDescription(offer)
	offerSignalSDP := SignalSDP{
		UserID: s.localPeerID,
		Signal: offer,
	}
	err = s.onSignalSDP(offerSignalSDP)
	return
}

func (s *Signaller) handleAnswer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("Error setting remote description: %w", err)
	}
	return nil
}
