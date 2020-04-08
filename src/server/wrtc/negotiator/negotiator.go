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
	AddTransceiverFromKind(codecType webrtc.RTPCodecType, init ...webrtc.RtpTransceiverInit) (*webrtc.RTPTransceiver, error)
}

type TransceiverRequest struct {
	CodecType webrtc.RTPCodecType
	Init      webrtc.RtpTransceiverInit
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

	queuedTransceiverRequests []TransceiverRequest
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

func (n *Negotiator) AddTransceiverFromKind(t TransceiverRequest) {
	n.mu.Lock()
	log.Printf("[%s] Added %s transceiver, direction: %s", n.remotePeerID, t.CodecType, t.Init.Direction)
	n.queuedTransceiverRequests = append(n.queuedTransceiverRequests, t)
	n.mu.Unlock()
	log.Printf("[%s] Calling Negotiate() because a new transceiver request request was received", n.remotePeerID)
	n.Negotiate()
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

func (n *Negotiator) addQueuedTransceivers() {
	for _, t := range n.queuedTransceiverRequests {
		log.Printf("[%s] add queued %s transceiver, direction: %s", n.remotePeerID, t.CodecType, t.Init.Direction)
		_, err := n.peerConnection.AddTransceiverFromKind(t.CodecType, t.Init)
		if err != nil {
			log.Printf("[%s] error adding %s transceiver: %s", n.remotePeerID, err)
		}
	}
	n.queuedTransceiverRequests = []TransceiverRequest{}
}

func (n *Negotiator) negotiate() {
	n.addQueuedTransceivers()

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
