package negotiator

import (
	"sync"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/pion/webrtc/v2"
)

var log = logger.GetLogger("negotiator")

type PeerConnection interface {
	CreateOffer(*webrtc.OfferOptions) (webrtc.SessionDescription, error)
	OnSignalingStateChange(func(webrtc.SignalingState))
}

type Negotiator struct {
	initiator            bool
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
	onOffer func(webrtc.SessionDescription, error),
	onRequestNegotiation func(),
) *Negotiator {
	n := &Negotiator{
		initiator:            initiator,
		peerConnection:       peerConnection,
		onOffer:              onOffer,
		onRequestNegotiation: onRequestNegotiation,
	}

	peerConnection.OnSignalingStateChange(n.handleSignalingStateChange)
	return n
}

func (n *Negotiator) handleSignalingStateChange(state webrtc.SignalingState) {
	// TODO check if we need to have a check for first stable state
	// like simple-peer has.
	log.Printf("Signaling state change: %s", state)

	if state == webrtc.SignalingStateStable {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.isNegotiating = false

		if n.queuedNegotiation {
			log.Printf("Executing queued negotiation")
			n.queuedNegotiation = false
			n.negotiate()
		}
	}
}

func (n *Negotiator) Negotiate() {
	log.Printf("Negotiate")

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.isNegotiating {
		n.queuedNegotiation = true
		return
	}

	n.isNegotiating = true
	n.negotiate()
}

func (n *Negotiator) negotiate() {
	if !n.initiator {
		log.Printf("Requesting renegotiation from initiator")
		n.requestNegotiation()
		return
	}

	log.Printf("Starting negotiation")
	offer, err := n.peerConnection.CreateOffer(nil)
	if err != nil {
		n.onOffer(offer, err)
		return
	}
	n.onOffer(offer, err)
}

func (n *Negotiator) requestNegotiation() {
	n.onRequestNegotiation()
}
