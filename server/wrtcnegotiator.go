package server

import (
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/pion/webrtc/v3"
)

type TransceiverRequest struct {
	CodecType webrtc.RTPCodecType
	Init      webrtc.RtpTransceiverInit
}

type Negotiator struct {
	log logger.Logger

	initiator            bool
	peerConnection       *webrtc.PeerConnection
	onOffer              func(webrtc.SessionDescription, error)
	onRequestNegotiation func()

	negotiationDone   chan struct{}
	mu                sync.Mutex
	queuedNegotiation bool

	queuedTransceiverRequests []TransceiverRequest
}

func NewNegotiator(
	log logger.Logger,
	initiator bool,
	peerConnection *webrtc.PeerConnection,
	onOffer func(webrtc.SessionDescription, error),
	onRequestNegotiation func(),
) *Negotiator {
	n := &Negotiator{
		log:                  log.WithNamespaceAppended("negotiator"),
		initiator:            initiator,
		peerConnection:       peerConnection,
		onOffer:              onOffer,
		onRequestNegotiation: onRequestNegotiation,
	}

	peerConnection.OnSignalingStateChange(n.handleSignalingStateChange)
	return n
}

func (n *Negotiator) AddTransceiverFromKind(t TransceiverRequest) {
	logCtx := logger.Ctx{
		"codec_type": t.CodecType,
		"direction":  t.Init.Direction,
	}

	n.log.Info("Add transceiver", logCtx)

	n.mu.Lock()
	n.queuedTransceiverRequests = append(n.queuedTransceiverRequests, t)
	n.mu.Unlock()

	n.log.Info("Negotiate because a transceiver was queued", logCtx)
	n.Negotiate()
}

func (n *Negotiator) closeDoneChannel() {
	if n.negotiationDone != nil {
		close(n.negotiationDone)
		n.negotiationDone = nil
	}
}

func (n *Negotiator) Done() <-chan struct{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.negotiationDone != nil {
		return n.negotiationDone
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (n *Negotiator) handleSignalingStateChange(state webrtc.SignalingState) {
	// TODO check if we need to have a check for first stable state
	// like simple-peer has.

	logCtx := logger.Ctx{
		"signaling_state": state,
	}

	n.log.Info("Signaling state changed", logCtx)

	n.mu.Lock()
	defer n.mu.Unlock()

	switch state {
	case webrtc.SignalingStateClosed:
		n.closeDoneChannel()
	case webrtc.SignalingStateStable:
		if n.queuedNegotiation {
			n.log.Info("Execute queued negotiation", logCtx)
			n.queuedNegotiation = false
			n.negotiate()
		} else {
			n.closeDoneChannel()
		}
	}
}

func (n *Negotiator) Negotiate() (done <-chan struct{}) {
	n.log.Info("Negotiate", nil)

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.negotiationDone != nil {
		n.log.Info("Negotiate: already negotiating, queueing for later", nil)
		n.queuedNegotiation = true
		return
	}

	n.log.Info("Negotiate: start", nil)
	n.negotiationDone = make(chan struct{})

	n.negotiate()
	return n.negotiationDone
}

func (n *Negotiator) addQueuedTransceivers() {
	for _, t := range n.queuedTransceiverRequests {
		logCtx := logger.Ctx{
			"codec_type": t.CodecType,
			"direction":  t.Init.Direction,
		}

		n.log.Trace("Add queued transceiver", logCtx)

		_, err := n.peerConnection.AddTransceiverFromKind(t.CodecType, t.Init)
		if err != nil {
			n.log.Error("Add queued transceiver", errors.Trace(err), logCtx)
		}
	}

	n.queuedTransceiverRequests = []TransceiverRequest{}
}

func (n *Negotiator) negotiate() {
	n.addQueuedTransceivers()

	if !n.initiator {
		n.log.Info("negotiate: requesting negotiation from initiator", nil)
		n.requestNegotiation()
		return
	}

	n.log.Info("negotiate: creating offer", nil)

	offer, err := n.peerConnection.CreateOffer(nil)

	n.onOffer(offer, errors.Annotate(err, "create offer"))
}

func (n *Negotiator) requestNegotiation() {
	n.onRequestNegotiation()
}
