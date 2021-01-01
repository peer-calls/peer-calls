package server

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/pion/webrtc/v3"
)

type Signaller struct {
	log logger.Logger

	peerConnection *webrtc.PeerConnection
	initiator      bool
	localPeerID    string
	remotePeerID   string
	negotiator     *Negotiator

	signalMu      sync.Mutex
	closed        bool
	signalChannel chan Payload
	closeChannel  chan struct{}
	closeOnce     sync.Once

	descriptionSent     chan struct{}
	descriptionSentOnce sync.Once
}

func NewSignaller(
	log logger.Logger,
	initiator bool,
	peerConnection *webrtc.PeerConnection,
	localPeerID string,
	remotePeerID string,
) (*Signaller, error) {
	log = log.WithNamespaceAppended("signaller")

	s := &Signaller{
		log:             log,
		initiator:       initiator,
		peerConnection:  peerConnection,
		localPeerID:     localPeerID,
		remotePeerID:    remotePeerID,
		signalChannel:   make(chan Payload),
		closeChannel:    make(chan struct{}),
		descriptionSent: make(chan struct{}),
	}

	negotiator := NewNegotiator(
		log,
		initiator,
		peerConnection,
		s.remotePeerID,
		s.handleLocalOffer,
		s.handleLocalRequestNegotiation,
	)

	s.negotiator = negotiator

	peerConnection.OnICEConnectionStateChange(s.handleICEConnectionStateChange)
	peerConnection.OnICECandidate(s.handleICECandidate)

	return s, errors.Annotate(s.initialize(), "new signaller")
}

// This does not close any channel, but returns a channel that can be used
// for signalling closing of peer connection
func (s *Signaller) Done() <-chan struct{} {
	return s.closeChannel
}

func (s *Signaller) SignalChannel() <-chan Payload {
	return s.signalChannel
}

func (s *Signaller) initialize() error {
	if s.initiator {
		s.log.Debug("Pre-add video transceiver", nil)
		_, err := s.peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeVideo,
			webrtc.RtpTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionRecvonly,
			},
		)
		if err != nil {
			return errors.Annotate(err, "add video transceiver")
		}

		s.log.Debug("Pre-add audio transceiver", nil)
		_, err = s.peerConnection.AddTransceiverFromKind(
			webrtc.RTPCodecTypeAudio,
			webrtc.RtpTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionRecvonly,
			},
		)
		if err != nil {
			return errors.Annotate(err, "add audio transceiver")
		}

		s.log.Info("Negotiate", nil)
		s.negotiator.Negotiate()
	}

	return nil
}

func (s *Signaller) Initiator() bool {
	return s.initiator
}

func (s *Signaller) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	s.log.Info("Peer connection state changed", logger.Ctx{
		"connection_state": connectionState,
	})

	if connectionState == webrtc.ICEConnectionStateClosed ||
		connectionState == webrtc.ICEConnectionStateDisconnected ||
		connectionState == webrtc.ICEConnectionStateFailed {
		s.Close()
	}
}

func (s *Signaller) onSignal(payload Payload) {
	s.signalMu.Lock()

	go func() {
		defer s.signalMu.Unlock()

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

		err = errors.Annotate(s.peerConnection.Close(), "close")
	})
	s.closeDescriptionSent()
	return
}

func (s *Signaller) handleICECandidate(c *webrtc.ICECandidate) {
	// wait until local description is set to prevent sending ice candidates
	// before local offer is sent
	s.log.Debug("Got ICE candidate (waiting)", nil)

	<-s.descriptionSent

	s.log.Debug("Got ICE candidate (processing)", nil)

	if c == nil {
		return
	}

	payload := Payload{
		UserID: s.localPeerID,
		Signal: Candidate{
			Candidate: c.ToJSON(),
		},
	}

	s.log.Debug("Got ICE candidate from server peer", logger.Ctx{
		"payload": payload,
	})

	s.onSignal(payload)
}

func (s *Signaller) Signal(payload map[string]interface{}) error {
	signalPayload, err := NewPayloadFromMap(payload)
	if err != nil {
		return errors.Annotate(err, "construct signal from payload")
	}

	switch signal := signalPayload.Signal.(type) {
	case Candidate:
		s.log.Debug("Remote candidate", logger.Ctx{
			"candidate": signal.Candidate.Candidate,
		})

		if signal.Candidate.Candidate != "" {
			return s.peerConnection.AddICECandidate(signal.Candidate)
		}
		return nil
	case Renegotiate:
		s.log.Trace("Remote peer wanted to negotiate", nil)
		s.Negotiate()
		return nil
	case TransceiverRequestPayload:
		s.log.Trace("Remote transceiver request", logger.Ctx{
			"transceiver_kind": signal.TransceiverRequest.Kind,
		})
		s.handleTransceiverRequest(signal)
		return nil
	case webrtc.SessionDescription:
		s.log.Trace("Remote sdp", logger.Ctx{
			"signal_type": signal.Type,
			"sdp":         signal.SDP,
		})
		return s.handleRemoteSDP(signal)
	default:
		return errors.Errorf("unexpected signal: %#v", signal)
	}
}

func (s *Signaller) handleTransceiverRequest(transceiverRequest TransceiverRequestPayload) {
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
		return errors.Annotate(s.handleRemoteOffer(sessionDescription), "handle remote offer")
	case webrtc.SDPTypeAnswer:
		return errors.Annotate(s.handleRemoteAnswer(sessionDescription), "handle remote answer")
	default:
		return errors.Errorf("unexpected sdp type: %s", sessionDescription.Type)
	}
}

func (s *Signaller) handleRemoteOffer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return errors.Annotate(err, "set remote description")
	}
	answer, err := s.peerConnection.CreateAnswer(nil)
	if err != nil {
		return errors.Annotate(err, "create answer")
	}
	if err := s.peerConnection.SetLocalDescription(answer); err != nil {
		return errors.Annotate(err, "set local description")
	}

	s.log.Trace("Local signal", logger.Ctx{
		"signal_type": answer.Type,
		"sdp":         answer.SDP,
	})

	s.onSignal(NewPayloadSDP(s.localPeerID, answer))

	// allow ice candidates to be sent
	s.closeDescriptionSent()
	return nil
}

func (s *Signaller) handleLocalRequestNegotiation() {
	s.log.Trace("Send negotiation request to initiator", nil)
	s.onSignal(NewPayloadRenegotiate(s.localPeerID))
}

func (s *Signaller) handleLocalOffer(offer webrtc.SessionDescription, err error) {
	if err != nil {
		s.log.Error("Local signal", errors.Trace(err), nil)
		// TODO abort connection
		return
	}

	s.log.Trace("Local signal", logger.Ctx{
		"signal_type": offer.Type,
		"sdp":         offer.SDP,
	})

	err = s.peerConnection.SetLocalDescription(offer)
	if err != nil {
		s.log.Error("Set local description", errors.Trace(err), nil)
		// TODO abort connection
		return
	}

	s.onSignal(NewPayloadSDP(s.localPeerID, offer))

	// allow ice candidates to be sent
	s.closeDescriptionSent()
}

// closeDescriptionSent closes the descriptionSent channel which allows the ICE
// candidates to be processed.
func (s *Signaller) closeDescriptionSent() {
	s.descriptionSentOnce.Do(func() {
		close(s.descriptionSent)
	})
}

// Sends a request for a new transceiver, only if the peer is not the initiator.
func (s *Signaller) SendTransceiverRequest(kind webrtc.RTPCodecType, direction webrtc.RTPTransceiverDirection) {
	if !s.initiator {
		s.log.Trace("Send transceiver request to initiator", nil)
		s.onSignal(NewTransceiverRequest(s.localPeerID, kind, direction))
	}
}

// TODO check offer voice activation detection feature of webrtc

// Create an offer and send it to remote peer
func (s *Signaller) Negotiate() <-chan struct{} {
	return s.negotiator.Negotiate()
}

func (s *Signaller) handleRemoteAnswer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return errors.Annotate(err, "set remote description")
	}

	return nil
}

// NegotiationDone returns the channel that will be closed as soon as the
// current negotiation is done. If there is no negotiation in progress, it
// returns a closed channel. If there is a negotiation in progress, and the
// negotiation was initiated by a call to Negotiate(), it will return the same
// channel as Negotiate.
func (s *Signaller) NegotiationDone() <-chan struct{} {
	return s.negotiator.Done()
}
