package signals

import (
	"fmt"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/jeremija/peer-calls/src/server-go/wrtc/negotiator"
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
	mediaEngine    *webrtc.MediaEngine
	initiator      bool
	localPeerID    string
	onSignal       func(signal interface{})
	negotiator     *negotiator.Negotiator
}

var log = logger.GetLogger("signals")

func NewSignaller(
	initiator bool,
	peerConnection PeerConnection,
	mediaEngine *webrtc.MediaEngine,
	localPeerID string,
	onSignal func(signal interface{}),
) (*Signaller, error) {
	s := Signaller{
		peerConnection: peerConnection,
		mediaEngine:    mediaEngine,
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

	if !initiator {
		_, err := peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeVideo,
			webrtc.RtpTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionRecvonly,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("Error adding video transceiver: %s", err)
		}
		// // TODO add one more video transceiver for screen sharing
		// // TODO add audio
		// _, err = peerConnection.AddTransceiverFromKind(
		// 	webrtc.RTPCodecTypeAudio,
		// 	webrtc.RtpTransceiverInit{
		// 		Direction: webrtc.RTPTransceiverDirectionRecvonly,
		// 	},
		// )
		// if err != nil {
		// 	log.Printf("Error adding audio transceiver: %s", err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }
	} else {
		log.Println("Registering default codecs")
		s.mediaEngine.RegisterDefaultCodecs()
		log.Println("Peer is initiator, calling Negotiate()")
		s.negotiator.Negotiate()
	}
	// peerConnection.OnICECandidate(s.handleICECandidate)

	return &s, nil
}

func (s *Signaller) Initiator() bool {
	return s.initiator
}

func (s *Signaller) handleICECandidate(c *webrtc.ICECandidate) {
	if c == nil {
		return
	}

	payload := Payload{
		UserID: s.localPeerID,
		Signal: Candidate{
			Candidate: c.ToJSON(),
		},
	}

	log.Printf("Got ice candidate from server peer: %s", payload)
	s.onSignal(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) error {
	signalPayload, err := NewPayloadFromMap(payload)

	if err != nil {
		return fmt.Errorf("Error constructing signal from payload: %s", err)
	}

	switch signal := signalPayload.Signal.(type) {
	case Candidate:
		log.Printf("Remote signal.canidate: %s", signal.Candidate)
		return s.peerConnection.AddICECandidate(signal.Candidate)
	case Renegotiate:
		log.Printf("Remote signal.renegotiate")
		log.Printf("Calling signaller.Negotiate() because remote peer wanted to negotiate")
		s.Negotiate()
		return nil
	case TransceiverRequest:
		log.Printf("Remote signal.transceiverRequest: %s", signal.TransceiverRequest.Kind)
		return s.handleTransceiverRequest(signal)
	case webrtc.SessionDescription:
		log.Printf("Remote signal.type: %s, signal.sdp: %s", signal.Type, signal.SDP)
		return s.handleRemoteSDP(signal)
	default:
		return fmt.Errorf("Unexpected signal: %#v", signal)
	}
}

func (s *Signaller) handleTransceiverRequest(transceiverRequest TransceiverRequest) (err error) {
	log.Printf("Got transceiver request %v", transceiverRequest)

	codecType := transceiverRequest.TransceiverRequest.Kind
	var t *webrtc.RTPTransceiver
	// if init := transceiverRequest.TransceiverRequest.Init; init != nil {
	// 	t, err = s.peerConnection.AddTransceiverFromKind(codecType, *init)
	// } else {
	t, err = s.peerConnection.AddTransceiverFromKind(
		codecType,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	// }
	log.Printf("Added %s transceiver, direction: %s", codecType, t.Direction())

	if err != nil {
		return fmt.Errorf("Error adding transceiver type %s: %s", codecType, err)
	}

	log.Printf("Calling signaller.Negotiate() because a new transceiver was added")
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
	if err = s.mediaEngine.PopulateFromSDP(sessionDescription); err != nil {
		return fmt.Errorf("Error populating codec info from SDP: %s", err)
	}

	log.Printf("Printing available codecs")
	for _, codec := range s.mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
		log.Println("codec", codec.PayloadType, codec.Name, codec.ClockRate)
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

	log.Printf("Local signal.type: %s, signal.sdp: %s", answer.Type, answer.SDP)
	s.onSignal(NewPayloadSDP(s.localPeerID, answer))
	return nil
}

func (s *Signaller) handleLocalRequestNegotiation() {
	log.Println("Sending renegotiation request to initiator")
	s.onSignal(NewPayloadRenegotiate(s.localPeerID))
}

func (s *Signaller) handleLocalOffer(offer webrtc.SessionDescription, err error) {
	log.Printf("Local signal.type: %s, signal.sdp: %s", offer.Type, offer.SDP)
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

	s.onSignal(NewPayloadSDP(s.localPeerID, offer))
}

// Sends a request for a new transceiver, only if the peer is not the initiator.
func (s *Signaller) SendTransceiverRequest(kind webrtc.RTPCodecType, direction webrtc.RTPTransceiverDirection) {
	if !s.initiator {
		log.Println("Sending transceiver request to initiator")
		s.onSignal(NewTransceiverRequest(s.localPeerID, kind, direction))
	}
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
