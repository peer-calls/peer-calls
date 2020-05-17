package server

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

type JitterHandler interface {
	HandleNack(clientID string, rtpSender *webrtc.RTPSender, nack *rtcp.TransportLayerNack) *rtcp.TransportLayerNack
	HandleRTP(clientID string, peerConnection *webrtc.PeerConnection, pkt *rtp.Packet)
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
func (n *NackHandler) HandleNack(clientID string, rtpSender *webrtc.RTPSender, nack *rtcp.TransportLayerNack) *rtcp.TransportLayerNack {
	actualNacks := make([]rtcp.NackPair, 0, len(nack.Nacks))

	for _, nackPair := range nack.Nacks {
		n.nackLog.Printf("[%s] NACK for track: %d (fsn: %d, blp: %#b)", clientID, nack.MediaSSRC, nackPair.PacketID, nackPair.LostPackets)

		nackPackets := nackPair.PacketList()
		notFound := make([]uint16, 0, len(nackPackets))
		for _, sn := range nackPackets {
			rtpPacket := n.jitterBuffer.GetPacket(nack.MediaSSRC, sn)
			if rtpPacket == nil {
				// missing packet not found in jitter buffer
				n.nackLog.Printf("[%s] Packet (ssrc: %d, sn: %d) missing", clientID, nack.MediaSSRC, sn)
				notFound = append(notFound, sn)
				continue
			}

			n.nackLog.Printf("[%s] Packet (ssrc: %d, sn: %d) found in JitterBuffer", clientID, nack.MediaSSRC, sn)
			// JitterBuffer had the missing packet, send it
			_, err := rtpSender.SendRTP(&rtpPacket.Header, rtpPacket.Payload)
			if err != nil {
				n.log.Printf("[%s] Error sending RTP packet from jitter buffer for track: %d: %s", clientID, nack.MediaSSRC, err)
			}
		}
		if len(notFound) > 0 {
			actualNacks = append(actualNacks, CreateNackPair(notFound))
		}
	}

	if len(actualNacks) == 0 {
		return nil
	}

	return &rtcp.TransportLayerNack{
		MediaSSRC:  nack.MediaSSRC,
		SenderSSRC: nack.SenderSSRC,
		Nacks:      actualNacks,
	}
}

func (n *NackHandler) HandleRTP(clientID string, peerConnection *webrtc.PeerConnection, pkt *rtp.Packet) {
	prometheusRTPPacketsReceived.Inc()
	rtcpPkt := n.jitterBuffer.PushRTP(pkt)
	if rtcpPkt == nil {
		return
	}

	err := peerConnection.WriteRTCP([]rtcp.Packet{rtcpPkt})
	if err != nil {
		n.log.Printf("[%s] Error writing rtcp packet from jitter buffer for track: %d: %s", clientID, pkt.SSRC, err)
	}
}

func (n *NackHandler) RemoveBuffer(ssrc uint32) {
	n.jitterBuffer.RemoveBuffer(ssrc)
}

type NoopNackHandler struct{}

func (n *NoopNackHandler) HandleNack(clientID string, rtpSender *webrtc.RTPSender, nack *rtcp.TransportLayerNack) *rtcp.TransportLayerNack {
	// do nothing
	return nil
}

func (n *NoopNackHandler) HandleRTP(clientID string, peerConnection *webrtc.PeerConnection, pkt *rtp.Packet) {
	// do nothing
}

func (n *NoopNackHandler) RemoveBuffer(ssrc uint32) {}
