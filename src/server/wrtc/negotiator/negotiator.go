package negotiator

import (
	"sync"

	"github.com/jeremija/peer-calls/src/server/logger"
	"github.com/pion/webrtc/v2"
)

var log = logger.GetLogger("negotiator")

type PeerConnection interface {
	CreateOffer(*webrtc.OfferOptions) (webrtc.SessionDescription, error)
	OnSignalingStateChange(func(webrtc.SignalingState))
}

type Negotiator struct {
	initiator            bool
	remotePeerID         string
	peerConnection       PeerConnection
	onOffer              func(webrtc.SessionDescription, error)
	onRequestNegotiation func()

	isNegotiating     bool
	mu                sync.Mutex
	queuedNegotiation bool
}

func NewNegotiator(
	initiator bool,
	peerConnection PeerConnection,
	remotePeerID string,
	onOffer func(webrtc.SessionDescription, error),
	onRequestNegotiation func(),
) *Negotiator {
	n := &Negotiator{
		initiator:            initiator,
		peerConnection:       peerConnection,
		remotePeerID:         remotePeerID,
		onOffer:              onOffer,
		onRequestNegotiation: onRequestNegotiation,
	}

	peerConnection.OnSignalingStateChange(n.handleSignalingStateChange)
	return n
}

func (n *Negotiator) handleSignalingStateChange(state webrtc.SignalingState) {
	// TODO check if we need to have a check for first stable state
	// like simple-peer has.
	log.Printf("[%s] Signaling state change for: %s", n.remotePeerID, state)

	if state == webrtc.SignalingStateStable {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.isNegotiating = false

		if n.queuedNegotiation {
			n.isNegotiating = true
			log.Printf("[%s] Executing queued negotiation", n.remotePeerID)
			n.queuedNegotiation = false
			n.negotiate()
		}
	}
}

func (n *Negotiator) Negotiate() {
	log.Printf("[%s] Negotiate", n.remotePeerID)

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.isNegotiating {
		log.Printf("[%s] Negotiate: already negotiating, queueing for later", n.remotePeerID)
		n.queuedNegotiation = true
		return
	}

	log.Printf("[%s] Negotiate: start", n.remotePeerID)
	n.isNegotiating = true
	n.negotiate()
}

func (n *Negotiator) negotiate() {
	if !n.initiator {
		log.Printf("[%s] negotiate: requesting from initiator", n.remotePeerID)
		n.requestNegotiation()
		return
	}

	log.Printf("[%s] negotiate: creating offer", n.remotePeerID)
	offer, err := n.peerConnection.CreateOffer(nil)
	n.onOffer(offer, err)
}

func (n *Negotiator) requestNegotiation() {
	n.onRequestNegotiation()
}
