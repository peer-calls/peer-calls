package server

import (
	"sync"

	"github.com/pion/webrtc/v2"
)

type TransceiverRequest struct {
	CodecType webrtc.RTPCodecType
	Init      webrtc.RtpTransceiverInit
}

type Negotiator struct {
	log Logger

	initiator            bool
	remotePeerID         string
	peerConnection       *webrtc.PeerConnection
	onOffer              func(webrtc.SessionDescription, error)
	onRequestNegotiation func()

	negotiationDone   chan struct{}
	mu                sync.Mutex
	queuedNegotiation bool

	queuedTransceiverRequests []TransceiverRequest
}

func NewNegotiator(
	loggerFactory LoggerFactory,
	initiator bool,
	peerConnection *webrtc.PeerConnection,
	remotePeerID string,
	onOffer func(webrtc.SessionDescription, error),
	onRequestNegotiation func(),
) *Negotiator {
	n := &Negotiator{
		log:                  loggerFactory.GetLogger("negotiator"),
		initiator:            initiator,
		peerConnection:       peerConnection,
		remotePeerID:         remotePeerID,
		onOffer:              onOffer,
		onRequestNegotiation: onRequestNegotiation,
	}

	peerConnection.OnSignalingStateChange(n.handleSignalingStateChange)
	return n
}

func (n *Negotiator) AddTransceiverFromKind(t TransceiverRequest) {
	n.mu.Lock()
	n.log.Printf("[%s] Queued %s transceiver, direction: %s", n.remotePeerID, t.CodecType, t.Init.Direction)
	n.queuedTransceiverRequests = append(n.queuedTransceiverRequests, t)
	n.mu.Unlock()
	n.log.Printf("[%s] Calling Negotiate because a %s transceiver was queued", n.remotePeerID, t.CodecType)
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
	n.log.Printf("[%s] Signaling state change: %s", n.remotePeerID, state)

	n.mu.Lock()
	defer n.mu.Unlock()

	switch state {
	case webrtc.SignalingStateClosed:
		n.closeDoneChannel()
	case webrtc.SignalingStateStable:
		if n.queuedNegotiation {
			n.log.Printf("[%s] Executing queued negotiation", n.remotePeerID)
			n.queuedNegotiation = false
			n.negotiate()
		} else {
			n.closeDoneChannel()
		}
	}
}

func (n *Negotiator) Negotiate() (done <-chan struct{}) {
	n.log.Printf("[%s] Negotiate", n.remotePeerID)

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.negotiationDone != nil {
		n.log.Printf("[%s] Negotiate: already negotiating, queueing for later", n.remotePeerID)
		n.queuedNegotiation = true
		return
	}

	n.log.Printf("[%s] Negotiate: start", n.remotePeerID)
	n.negotiationDone = make(chan struct{})

	n.negotiate()
	return n.negotiationDone
}

func (n *Negotiator) addQueuedTransceivers() {
	for _, t := range n.queuedTransceiverRequests {
		n.log.Printf("[%s] Adding queued %s transceiver, direction: %s", n.remotePeerID, t.CodecType, t.Init.Direction)
		_, err := n.peerConnection.AddTransceiverFromKind(t.CodecType, t.Init)
		if err != nil {
			n.log.Printf("[%s] Error adding %s transceiver: %s", n.remotePeerID, err)
		}
	}
	n.queuedTransceiverRequests = []TransceiverRequest{}
}

func (n *Negotiator) negotiate() {
	n.addQueuedTransceivers()

	if !n.initiator {
		n.log.Printf("[%s] negotiate: requesting from initiator", n.remotePeerID)
		n.requestNegotiation()
		return
	}

	n.log.Printf("[%s] negotiate: creating offer", n.remotePeerID)
	offer, err := n.peerConnection.CreateOffer(nil)
	n.onOffer(offer, err)
}

func (n *Negotiator) requestNegotiation() {
	n.onRequestNegotiation()
}
