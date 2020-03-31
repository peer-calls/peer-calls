package wrtc

import (
	"fmt"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/negotiator"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/signals"
	"github.com/pion/webrtc/v2"
)

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

	payload := signals.Payload{
		UserID: s.localPeerID,
		Signal: signals.Candidate{
			Candidate: c.ToJSON(),
		},
	}

	log.Printf("Got ice candidate from server peer: %s", payload)
	s.onSignal(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) error {
	signalPayload, err := signals.NewPayloadFromMap(payload)

	if err != nil {
		return fmt.Errorf("Error constructing signal from payload: %s", err)
	}

	switch signal := signalPayload.Signal.(type) {
	case signals.Candidate:
		return s.peerConnection.AddICECandidate(signal.Candidate)
	case signals.Renegotiate:
		s.Negotiate()
		return nil
	case signals.TransceiverRequest:
		return s.handleTransceiverRequest(signal)
	case webrtc.SessionDescription:
		return s.handleRemoteSDP(signal)
	default:
		return fmt.Errorf("Unexpected signal: %#v", signal)
	}
}

func (s *Signaller) handleTransceiverRequest(transceiverRequest signals.TransceiverRequest) (err error) {
	log.Printf("Got transceiver request %s", transceiverRequest)

	codecType := transceiverRequest.TransceiverRequest.Kind
	if init := transceiverRequest.TransceiverRequest.Init; init != nil {
		_, err = s.peerConnection.AddTransceiverFromKind(codecType, *init)
	} else {
		_, err = s.peerConnection.AddTransceiverFromKind(codecType)
	}

	if err != nil {
		return fmt.Errorf("Error adding transceiver type %s: %s", codecType, err)
	}

	s.Negotiate()
	return nil
}

func (s *Signaller) handleRemoteSDP(sessionDescription webrtc.SessionDescription) (err error) {
	switch sessionDescription.Type {
	case webrtc.SDPTypeOffer:
		return s.handleRemoteOffer(sessionDescription)
	case webrtc.SDPTypeAnswer:
		return s.handleRemoteAnswer(sessionDescription)
	default:
		return fmt.Errorf("Unexpected sdp type: %s", sessionDescription.Type)
	}
}

func (s *Signaller) handleRemoteOffer(sessionDescription webrtc.SessionDescription) (err error) {
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

	// log.Printf("Sending answer: %#v", answerSignalSDP)
	s.onSignal(signals.NewPayloadSDP(s.localPeerID, answer))
	return nil
}

func (s *Signaller) handleLocalRequestNegotiation() {
	log.Println("Sending renegotiation request to initiator")
	s.onSignal(signals.NewPayloadRenegotiate(s.localPeerID))
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

	s.onSignal(signals.NewPayloadSDP(s.localPeerID, offer))
}

// TODO check offer voice activation detection feature of webrtc

// Create an offer and send it to remote peer
func (s *Signaller) Negotiate() {
	s.negotiator.Negotiate()
}

func (s *Signaller) handleRemoteAnswer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("Error setting remote description: %w", err)
	}
	return nil
}
