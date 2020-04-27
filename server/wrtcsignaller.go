package server

import (
	"fmt"
	"sync"

	"github.com/pion/webrtc/v2"
)

const IOSH264Fmtp = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"

type Signaller struct {
	log    Logger
	sdpLog Logger

	peerConnection *webrtc.PeerConnection
	mediaEngine    *webrtc.MediaEngine
	initiator      bool
	localPeerID    string
	remotePeerID   string
	negotiator     *Negotiator

	signalMu      sync.RWMutex
	closed        bool
	signalChannel chan Payload
	closeChannel  chan struct{}
	closeOnce     sync.Once
}

func NewSignaller(
	loggerFactory LoggerFactory,
	initiator bool,
	peerConnection *webrtc.PeerConnection,
	mediaEngine *webrtc.MediaEngine,
	localPeerID string,
	remotePeerID string,
) (*Signaller, error) {
	s := &Signaller{
		log:            loggerFactory.GetLogger("signaller"),
		sdpLog:         loggerFactory.GetLogger("sdp"),
		initiator:      initiator,
		peerConnection: peerConnection,
		mediaEngine:    mediaEngine,
		localPeerID:    localPeerID,
		remotePeerID:   remotePeerID,
		signalChannel:  make(chan Payload),
		closeChannel:   make(chan struct{}),
	}

	negotiator := NewNegotiator(
		loggerFactory,
		initiator,
		peerConnection,
		s.remotePeerID,
		s.handleLocalOffer,
		s.handleLocalRequestNegotiation,
	)

	s.negotiator = negotiator

	peerConnection.OnICEConnectionStateChange(s.handleICEConnectionStateChange)
	// peerConnection.OnICECandidate(s.handleICECandidate)

	return s, s.initialize()
}

// This does not close any channel, but returns a channel that can be used
// for signalling closing of peer connection
func (s *Signaller) CloseChannel() <-chan struct{} {
	return s.closeChannel
}

func (s *Signaller) SignalChannel() <-chan Payload {
	return s.signalChannel
}

func (s *Signaller) initialize() error {
	if s.initiator {
		s.log.Printf("[%s] NewSignaller: Initiator registering default codecs", s.remotePeerID)
		// s.mediaEngine.RegisterDefaultCodecs()
		s.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

		rtcpfb := []webrtc.RTCPFeedback{
			webrtc.RTCPFeedback{
				Type: webrtc.TypeRTCPFBGoogREMB,
			},
			webrtc.RTCPFeedback{
				Type: webrtc.TypeRTCPFBCCM,
			},
			webrtc.RTCPFeedback{
				Type: webrtc.TypeRTCPFBNACK,
			},
			webrtc.RTCPFeedback{
				Type: "nack pli",
			},
		}

		s.mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
		// s.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
		// s.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
	}

	s.log.Printf("[%s] NewSignaller: Non-Initiator pre-add video transceiver", s.remotePeerID)
	_, err := s.peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		s.log.Printf("ERROR: %s", err)
		return fmt.Errorf("[%s] NewSignaller: Error pre-adding video transceiver: %s", s.remotePeerID, err)
	}

	s.log.Printf("[%s] NewSignaller: Non-Initiator pre-add audio transceiver", s.remotePeerID)
	_, err = s.peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		return fmt.Errorf("[%s] NewSignaller: Error pre-adding audio transceiver: %s", s.remotePeerID, err)
	}

	if s.initiator {
		s.log.Printf("[%s] NewSignaller: Initiator calling Negotiate()", s.remotePeerID)
		s.negotiator.Negotiate()
	}

	return nil
}

func (s *Signaller) Initiator() bool {
	return s.initiator
}

func (s *Signaller) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	s.log.Printf("[%s] Peer connection state changed: %s", s.remotePeerID, connectionState.String())
	if connectionState == webrtc.ICEConnectionStateClosed ||
		connectionState == webrtc.ICEConnectionStateDisconnected ||
		connectionState == webrtc.ICEConnectionStateFailed {
		s.Close()
	}

}

func (s *Signaller) onSignal(payload Payload) {
	s.signalMu.RLock()

	go func() {
		defer s.signalMu.RUnlock()

		ch := s.signalChannel
		if s.closed {
			// read from nil channel blocks indefinitely
			ch = nil
		}

		select {
		case ch <- payload:
			// successfully sent
		case <-s.closeChannel:
			// signaller has been closed
			return
		}
	}()
}

func (s *Signaller) Close() (err error) {
	s.closeOnce.Do(func() {
		close(s.closeChannel)

		s.signalMu.Lock()
		defer s.signalMu.Unlock()

		close(s.signalChannel)
		s.closed = true

		err = s.peerConnection.Close()
	})
	return
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

	s.log.Printf("[%s] Got ice candidate from server peer: %s", payload, s.remotePeerID)
	s.onSignal(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) error {
	signalPayload, err := NewPayloadFromMap(payload)

	if err != nil {
		return fmt.Errorf("Error constructing signal from payload: %s", err)
	}

	switch signal := signalPayload.Signal.(type) {
	case Candidate:
		s.log.Printf("[%s] Remote signal.canidate: %s ", signal.Candidate, s.remotePeerID)
		return s.peerConnection.AddICECandidate(signal.Candidate)
	case Renegotiate:
		s.log.Printf("[%s] Remote signal.renegotiate ", s.remotePeerID)
		s.log.Printf("[%s] Calling signaller.Negotiate() because remote peer wanted to negotiate", s.remotePeerID)
		s.Negotiate()
		return nil
	case TransceiverRequestPayload:
		s.log.Printf("[%s] Remote signal.transceiverRequest: %s", s.remotePeerID, signal.TransceiverRequest.Kind)
		s.handleTransceiverRequest(signal)
		return nil
	case webrtc.SessionDescription:
		s.sdpLog.Printf("[%s] Remote signal.type: %s, signal.sdp: %s", s.remotePeerID, signal.Type, signal.SDP)
		return s.handleRemoteSDP(signal)
	default:
		return fmt.Errorf("[%s] Unexpected signal: %#v ", s.remotePeerID, signal)
	}
}

func (s *Signaller) handleTransceiverRequest(transceiverRequest TransceiverRequestPayload) {
	s.log.Printf("[%s] handleTransceiverRequest: %v", s.remotePeerID, transceiverRequest)

	codecType := transceiverRequest.TransceiverRequest.Kind

	s.negotiator.AddTransceiverFromKind(TransceiverRequest{
		CodecType: codecType,
		Init: webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendrecv,
		},
	})
}

func (s *Signaller) handleRemoteSDP(sessionDescription webrtc.SessionDescription) (err error) {
	switch sessionDescription.Type {
	case webrtc.SDPTypeOffer:
		return s.handleRemoteOffer(sessionDescription)
	case webrtc.SDPTypeAnswer:
		return s.handleRemoteAnswer(sessionDescription)
	default:
		return fmt.Errorf("[%s] Unexpected sdp type: %s", s.remotePeerID, sessionDescription.Type)
	}
}

func (s *Signaller) handleRemoteOffer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.mediaEngine.PopulateFromSDP(sessionDescription); err != nil {
		return fmt.Errorf("[%s] Error populating codec info from SDP: %s", s.remotePeerID, err)
	}

	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("[%s] Error setting remote description: %w", s.remotePeerID, err)
	}
	answer, err := s.peerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("[%s] Error creating answer: %w", s.remotePeerID, err)
	}
	if err := s.peerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("[%s] Error setting local description: %w", s.remotePeerID, err)
	}

	s.sdpLog.Printf("[%s] Local signal.type: %s, signal.sdp: %s", s.remotePeerID, answer.Type, answer.SDP)
	s.onSignal(NewPayloadSDP(s.localPeerID, answer))
	return nil
}

func (s *Signaller) handleLocalRequestNegotiation() {
	s.log.Printf("[%s] Sending renegotiation request to initiator", s.remotePeerID)
	s.onSignal(NewPayloadRenegotiate(s.localPeerID))
}

func (s *Signaller) handleLocalOffer(offer webrtc.SessionDescription, err error) {
	s.sdpLog.Printf("[%s] Local signal.type: %s, signal.sdp: %s", s.remotePeerID, offer.Type, offer.SDP)
	if err != nil {
		s.log.Printf("[%s] Error creating local offer: %s", s.remotePeerID, err)
		// TODO abort connection
		return
	}

	err = s.peerConnection.SetLocalDescription(offer)
	if err != nil {
		s.log.Printf("[%s] Error setting local description from local offer: %s", s.remotePeerID, err)
		// TODO abort connection
		return
	}

	s.onSignal(NewPayloadSDP(s.localPeerID, offer))
}

// Sends a request for a new transceiver, only if the peer is not the initiator.
func (s *Signaller) SendTransceiverRequest(kind webrtc.RTPCodecType, direction webrtc.RTPTransceiverDirection) {
	if !s.initiator {
		s.log.Printf("[%s] Sending transceiver request to initiator", s.remotePeerID)
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
		return fmt.Errorf("[%s] Error setting remote description: %w", s.remotePeerID, err)
	}
	return nil
}
