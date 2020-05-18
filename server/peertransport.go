package server

import (
	"fmt"
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type PeerTransport struct {
	mu sync.Mutex

	peerConnection  *webrtc.PeerConnection
	peerInfoByTrack map[uint32]peerTrackInfo
}

type peerTrackInfo struct {
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
}

func NewPeerTransport(peerConnection *webrtc.PeerConnection) *PeerTransport {
	return &PeerTransport{
		peerConnection:  peerConnection,
		peerInfoByTrack: map[uint32]peerTrackInfo{},
	}
}

func (p *PeerTransport) WriteRTCP(packet []rtcp.Packet) error {
	return p.peerConnection.WriteRTCP(packet)
}

func (p *PeerTransport) WriteRTP(packet *rtp.Packet) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	i, ok := p.peerInfoByTrack[packet.SSRC]
	if !ok {
		return 0, fmt.Errorf("WriteRTP: track with SSRC not found: %d", packet.SSRC)
	}

	return i.sender.SendRTP(&packet.Header, packet.Payload)
}

func (p *PeerTransport) NewTrack(payloadType uint8, ssrc uint32, id string, label string) (*webrtc.Track, error) {
	return p.peerConnection.NewTrack(payloadType, ssrc, id, label)
}

func (p *PeerTransport) AddTrack(track *webrtc.Track) (<-chan rtcp.Packet, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rtcpCh := make(chan rtcp.Packet)
	sender, err := p.peerConnection.AddTrack(track)

	if err != nil {
		close(rtcpCh)
		return rtcpCh, err
	}

	go func() {
		defer close(rtcpCh)
		for {
			rtcpPackets, err := sender.ReadRTCP()
			if err != nil {
				return
			}
			for _, rtcpPacket := range rtcpPackets {
				rtcpCh <- rtcpPacket
			}
		}
	}()

	var transceiver *webrtc.RTPTransceiver
	for _, tr := range p.peerConnection.GetTransceivers() {
		if tr.Sender() == sender {
			transceiver = tr
			break
		}
	}

	p.peerInfoByTrack[track.SSRC()] = peerTrackInfo{
		sender:      sender,
		transceiver: transceiver,
	}

	return rtcpCh, err
}

func (p *PeerTransport) RemoveTrack(track *webrtc.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	i, ok := p.peerInfoByTrack[track.SSRC()]
	if !ok {
		return fmt.Errorf("sender not found for track: %d", track.SSRC())
	}
	return p.peerConnection.RemoveTrack(i.sender)
}

func (p *PeerTransport) OnTrack(hdlr func(*webrtc.Track)) {
	p.peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		hdlr(track)
	})
}

func (p *PeerTransport) Mid(ssrc uint32) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	i, ok := p.peerInfoByTrack[ssrc]
	if !ok {
		return ""
	}
	return i.transceiver.Mid()
}

var _ Transport = &PeerTransport{}
