package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type JitterHandler interface {
	HandleNack(nack *rtcp.TransportLayerNack) ([]*rtp.Packet, *rtcp.TransportLayerNack)
	HandleRTP(pkt *rtp.Packet) rtcp.Packet
	RemoveBuffer(ssrc uint32)
}

type NackHandler struct {
	log          Logger
	nackLog      Logger
	jitterBuffer *JitterBuffer
}

func NewJitterHandler(log Logger, nackLog Logger, enabled bool) JitterHandler {
	if enabled {
		return NewJitterNackHandler(log, nackLog, NewJitterBuffer())
	}
	return &NoopNackHandler{}
}

func NewJitterNackHandler(
	log Logger,
	nackLog Logger,
	jitterBuffer *JitterBuffer,
) *NackHandler {
	return &NackHandler{log, nackLog, jitterBuffer}
}

// ProcessNack tries to find the missing packet in JitterBuffer and send it,
// otherwise it will send the nack packet to the original sender of the track.
func (n *NackHandler) HandleNack(nack *rtcp.TransportLayerNack) ([]*rtp.Packet, *rtcp.TransportLayerNack) {
	actualNacks := make([]rtcp.NackPair, 0, len(nack.Nacks))

	var rtpPackets []*rtp.Packet

	for _, nackPair := range nack.Nacks {
		n.nackLog.Printf("NACK for track: %d (fsn: %d, blp: %#b)", nack.MediaSSRC, nackPair.PacketID, nackPair.LostPackets)

		nackPackets := nackPair.PacketList()
		notFound := make([]uint16, 0, len(nackPackets))
		for _, sn := range nackPackets {
			rtpPacket := n.jitterBuffer.GetPacket(nack.MediaSSRC, sn)
			if rtpPacket == nil {
				// missing packet not found in jitter buffer
				n.nackLog.Printf("RTP packet (ssrc: %d, sn: %d) missing", nack.MediaSSRC, sn)
				notFound = append(notFound, sn)
				continue
			}

			n.nackLog.Printf("RTP packet (ssrc: %d, sn: %d) found in JitterBuffer", nack.MediaSSRC, sn)

			// JitterBuffer had the missing packet, add it to the list
			rtpPackets = append(rtpPackets, rtpPacket)
		}
		if len(notFound) > 0 {
			actualNacks = append(actualNacks, CreateNackPair(notFound))
		}
	}

	if len(actualNacks) == 0 {
		return rtpPackets, nil
	}

	return rtpPackets, &rtcp.TransportLayerNack{
		MediaSSRC:  nack.MediaSSRC,
		SenderSSRC: nack.SenderSSRC,
		Nacks:      actualNacks,
	}
}

func (n *NackHandler) HandleRTP(pkt *rtp.Packet) rtcp.Packet {
	return n.jitterBuffer.PushRTP(pkt)
}

func (n *NackHandler) RemoveBuffer(ssrc uint32) {
	n.jitterBuffer.RemoveBuffer(ssrc)
}

type NoopNackHandler struct{}

func (n *NoopNackHandler) HandleNack(nack *rtcp.TransportLayerNack) ([]*rtp.Packet, *rtcp.TransportLayerNack) {
	// do nothing
	return nil, nil
}

func (n *NoopNackHandler) HandleRTP(pkt *rtp.Packet) rtcp.Packet {
	return nil
}

func (n *NoopNackHandler) RemoveBuffer(ssrc uint32) {}
