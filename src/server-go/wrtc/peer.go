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

type Signaller struct {
	peerConnection    *webrtc.PeerConnection
	localPeerID       string
	onSignalSDP       func(signal SignalSDP) error
	onSignalCandidate func(signal SignalCandidate)
}

var log = logger.GetLogger("wrtc")

func NewSignaller(
	peerConnection *webrtc.PeerConnection,
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

	peerConnection.OnICEConnectionStateChange(s.handleICEConnectionStateChange)
	peerConnection.OnICECandidate(s.handleICECandidate)

	return &s
}

func (s *Signaller) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	log.Printf("Peer connection state changed %s", connectionState.String())
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

	log.Printf("Got ice candidate from sever peer: %s", payload)
	s.onSignalCandidate(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) (err error) {
	signal, _ := payload["signal"].(map[string]interface{})
	remotePeerID, _ := payload["userId"].(string)

	if remotePeerID != s.localPeerID {
		return fmt.Errorf("Peer2Server only sends signals to server as peer")
	}

	if candidate, ok := signal["candidate"]; ok {
		err = s.handleSignalCandidate(remotePeerID, candidate)
	} else if sdpTypeString, ok := signal["type"]; ok {
		err = s.handleSDP(sdpTypeString, signal["sdp"])
	} else {
		err = fmt.Errorf("Unexpected signal message")
	}

	return
}

func (s *Signaller) handleSignalCandidate(targetClientID string, candidate interface{}) (err error) {
	log.Printf("Got client ice candidate: %s", candidate)
	candidateString, ok := candidate.(string)
	if !ok {
		return fmt.Errorf("Expected ice candidate to be staring")
	}
	iceCandidate := webrtc.ICECandidateInit{Candidate: candidateString}
	err = s.peerConnection.AddICECandidate(iceCandidate)
	return
}

func (s *Signaller) handleSDP(sdpType interface{}, sdp interface{}) (err error) {
	sdpTypeString, _ := sdpType.(string)
	sdpString, _ := sdp.(string)
	sessionDescription := webrtc.SessionDescription{}
	sessionDescription.SDP = sdpString
	log.Printf("Got client signal: %s", sdp)

	switch sdpTypeString {
	case "offer":
		sessionDescription.Type = webrtc.SDPTypeOffer
		// mediaEngine.PopulateFromSDP(sdp) TODO figure out if we need this
		// videoCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
		// audioCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)
	case "answer":
		sessionDescription.Type = webrtc.SDPTypeAnswer
	case "pranswer":
		sessionDescription.Type = webrtc.SDPTypePranswer
	case "rollback":
		sessionDescription.Type = webrtc.SDPTypeRollback
	default:
		return fmt.Errorf("Unknown sdp type: %s", sdpString)
	}

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

	err = s.onSignalSDP(SignalSDP{
		UserID: s.localPeerID,
		Signal: answer,
	})
	return err
}
