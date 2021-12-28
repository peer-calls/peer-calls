package sfu

import (
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type JitterHandler interface {
	HandleNack(nack *rtcp.TransportLayerNack) ([]*rtp.Packet, *rtcp.TransportLayerNack)
	HandleRTP(pkt *rtp.Packet) rtcp.Packet
	RemoveBuffer(ssrc uint32)
}

type NackHandler struct {
	log          logger.Logger
	nackLog      logger.Logger
	jitterBuffer *JitterBuffer
}

func NewJitterHandler(log logger.Logger, enabled bool) JitterHandler {
	if enabled {
		return NewJitterNackHandler(log, NewJitterBuffer())
	}

	return &NoopNackHandler{}
}

func NewJitterNackHandler(
	log logger.Logger,
	jitterBuffer *JitterBuffer,
) *NackHandler {
	log = log.WithNamespaceAppended("jitter")

	return &NackHandler{
		log:          log,
		nackLog:      log.WithNamespaceAppended("nack"),
		jitterBuffer: jitterBuffer,
	}
}

// ProcessNack tries to find the missing packet in JitterBuffer and send it,
// otherwise it will send the nack packet to the original sender of the track.
func (n *NackHandler) HandleNack(nack *rtcp.TransportLayerNack) ([]*rtp.Packet, *rtcp.TransportLayerNack) {
	actualNacks := make([]rtcp.NackPair, 0, len(nack.Nacks))

	var rtpPackets []*rtp.Packet

	for _, nackPair := range nack.Nacks {
		n.nackLog.Info("NACK for track: %d (fsn: %d, blp: %#b)", logger.Ctx{
			"ssrc": nack.MediaSSRC,
			"fsn":  nackPair.PacketID,
			"blp":  nackPair.LostPackets,
		})

		nackPackets := nackPair.PacketList()
		notFound := make([]uint16, 0, len(nackPackets))

		for _, sn := range nackPackets {
			rtpPacket := n.jitterBuffer.GetPacket(nack.MediaSSRC, sn)
			if rtpPacket == nil {
				// missing packet not found in jitter buffer
				n.nackLog.Info("RTP packet (ssrc: %d, sn: %d) missing", logger.Ctx{
					"ssrc": nack.MediaSSRC,
					"sn":   sn,
				})

				notFound = append(notFound, sn)

				continue
			}

			n.nackLog.Info("RTP packet (ssrc: %d, sn: %d) found in JitterBuffer", logger.Ctx{
				"ssrc": nack.MediaSSRC,
				"sn":   sn,
			})

			// JitterBuffer had the missing packet, add it to the list
			// FIXME https://github.com/peer-calls/peer-calls/issues/185
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
