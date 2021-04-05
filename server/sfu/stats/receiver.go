package stats

import (
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type Receiver struct {
	mu      sync.RWMutex
	members map[uint32]struct{}
	senders map[uint32]struct{}
}

func NewReceiver() *Receiver {
	return &Receiver{
		members: map[uint32]struct{}{},
		senders: map[uint32]struct{}{},
	}
}

func (r *Receiver) ReceiveRTP(packet *rtp.Packet) {
	r.mu.Lock()

	r.members[packet.SSRC] = struct{}{}
	r.senders[packet.SSRC] = struct{}{}

	r.mu.Unlock()
}

func (r *Receiver) handleReceptionReports(rr []rtcp.ReceptionReport) {
	r.mu.Lock()

	for _, report := range rr {
		r.members[report.SSRC] = struct{}{}
	}

	r.mu.Unlock()
}

func (r *Receiver) handleBye(bye *rtcp.Goodbye) {
	r.mu.Lock()

	for _, ssrc := range bye.Sources {
		delete(r.members, ssrc)
		delete(r.senders, ssrc)
	}

	r.mu.Unlock()
}

func (r *Receiver) ReceiveRTCP(packet rtcp.Packet) {
	switch p := packet.(type) {
	case *rtcp.ReceiverReport:
		r.handleReceptionReports(p.Reports)
	case *rtcp.SenderReport:
		r.handleReceptionReports(p.Reports)
	case *rtcp.Goodbye:
		r.handleBye(p)
	}
}

func (r *Receiver) Stats() (members, senders int) {
	r.mu.RLock()

	members = len(r.members)
	senders = len(r.senders)

	r.mu.RUnlock()

	return
}
