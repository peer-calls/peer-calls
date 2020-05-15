package server

import (
	"math"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// Thanks to MIT-licensed pion/ion for inspiration!

// maxSN is a maximum sequence number
const maxSN = uint16(math.MaxUint16)

// videoClock represents the clock rate of VP8, VP9 and H264 codecs (90000 Hz)
const videoClock = 90000

// keep only packets relevant to last 2s of video
const maxBufferTSDelta = videoClock * 2

//1+16(FSN+BLP) https://tools.ietf.org/html/rfc2032#page-9
const maxNackPairSize uint16 = 17

// Buffer holds the recent RTP packets and creates NACK RTCP packets
type Buffer struct {
	packets [int(maxSN) + 1]*rtp.Packet

	initialized bool

	lastPushSN  uint16
	lastNackSN  uint16
	lastClearTS uint32
	lastClearSN uint16

	nackWindowSize uint16

	ssrc uint32
}

// NewBuffer creates a new buffer for recent RTP packets
func NewBuffer() *Buffer {
	var b Buffer
	b.nackWindowSize = maxNackPairSize
	return &b
}

// snDelta calculates the distance between the start and end when using the
// ring buffer.
func snDelta(startSN uint16, endSN uint16) uint16 {
	return endSN - startSN
}

func tsDelta(ts1, ts2 uint32) uint32 {
	if ts1 > ts2 {
		return ts1 - ts2
	}
	return ts2 - ts1
}

func (b *Buffer) Push(p *rtp.Packet) rtcp.Packet {
	sn := p.SequenceNumber

	if !b.initialized {
		b.lastNackSN = sn
		b.lastClearTS = p.Timestamp
		// set it to one less because otherwise this packet would stay in memory
		// until the next cycle over the buffer
		b.lastClearSN = sn - 1

		b.ssrc = p.SSRC

		b.initialized = true
	}

	b.packets[sn] = p
	b.lastPushSN = sn

	b.clearOldPackets(p.Timestamp, sn)

	isNackReportWindow := sn-b.lastNackSN >= b.nackWindowSize
	// limit nack range
	if isNackReportWindow {
		windowStart := sn - b.nackWindowSize
		windowEnd := sn

		nackPairs, lostPkts := b.getNackPairs(windowStart, windowEnd)
		b.lastNackSN = sn
		if lostPkts > 0 {
			return &rtcp.TransportLayerNack{
				//origin ssrc
				SenderSSRC: b.ssrc,
				MediaSSRC:  b.ssrc,
				Nacks:      nackPairs,
			}
		}
	}

	return nil
}

func (b *Buffer) GetPacket(sequenceNumber uint16) *rtp.Packet {
	return b.packets[sequenceNumber]
}

func (b *Buffer) SSRC() uint32 {
	return b.ssrc
}

func (b *Buffer) getNackPairs(start uint16, end uint16) ([]rtcp.NackPair, int) {
	delta := end - start
	size := delta / maxNackPairSize
	arraySize := size
	if delta%maxNackPairSize > 0 {
		arraySize += 1
	}

	var totalLostPkt int
	pairs := make([]rtcp.NackPair, 0, arraySize)
	for i := uint16(0); i < size; i++ {
		nackPair, lostPkt := b.getNackPair(start, start+maxNackPairSize)
		totalLostPkt += lostPkt
		if lostPkt > 0 {
			pairs = append(pairs, nackPair)
		}
		start += maxNackPairSize
	}

	if arraySize > size {
		nackPair, lostPkt := b.getNackPair(start, end)
		totalLostPkt += lostPkt
		if lostPkt > 0 {
			pairs = append(pairs, nackPair)
		}
	}

	return pairs, totalLostPkt
}

// getNackPair returns the information about lost packets in the last nack
// window. The delta between start and end should be equal to maxNackPairSize.
func (b *Buffer) getNackPair(start uint16, end uint16) (rtcp.NackPair, int) {
	var lostPkts int

	// first sequence number lost
	var fsn uint16

	// Bitmask of following lost packets (BLP).
	// A bit is set to 1 if the corresponding packet has been lost,
	// and set to 0 otherwise. BLP is set to 0 only if no packet
	// other than that being NACKed (using the FSN field) has been
	// lost. BLP is set to 0x00001 if the packet corresponding to
	// the FSN and the following packet have been lost, etc.
	var blp rtcp.PacketBitmap

	// use != instead of < in case the sequence number has overflown
	for i := start; i != end; i++ {
		if b.packets[i] == nil {
			fsn = i
			lostPkts++
			break
		}
	}

	if lostPkts == 0 {
		return rtcp.NackPair{}, lostPkts
	}

	for i := fsn + 1; i != end; i++ {
		if b.packets[i] == nil {
			blp = AddBLP(fsn, i, blp)
			lostPkts++
		}
	}

	return rtcp.NackPair{PacketID: fsn, LostPackets: blp}, lostPkts
}

// clearOldPackets clears old packets by timestamp
func (b *Buffer) clearOldPackets(ts uint32, sn uint16) {
	clearTS := b.lastClearTS
	clearSN := b.lastClearSN

	if tsDelta(ts, clearTS) >= maxBufferTSDelta {
		for i := clearSN + 1; i != sn; i++ {
			pkt := b.packets[i]
			if pkt == nil {
				continue
			}
			if tsDelta(ts, pkt.Timestamp) < maxBufferTSDelta {
				// we've reached newer packets we want to keep, abort
				break
			}

			b.lastClearTS = pkt.Timestamp
			b.lastClearSN = i
			b.packets[i] = nil
		}
	}
}

func calcBLP(fsn uint16, missingSN uint16) rtcp.PacketBitmap {
	return (1 << (missingSN - fsn - 1))
}

func AddBLP(fsn uint16, missingSN uint16, blp rtcp.PacketBitmap) rtcp.PacketBitmap {
	return blp | calcBLP(fsn, missingSN)
}

func SubBLP(fsn uint16, foundSN uint16, blp rtcp.PacketBitmap) rtcp.PacketBitmap {
	return blp & ^calcBLP(fsn, foundSN)
}

func CreateNackPair(sequenceNumbers []uint16) rtcp.NackPair {
	lostPkts := len(sequenceNumbers)
	if lostPkts == 0 {
		return rtcp.NackPair{}
	}

	fsn := sequenceNumbers[0]
	var blp rtcp.PacketBitmap

	for _, i := range sequenceNumbers[1:] {
		blp = AddBLP(fsn, i, blp)
	}

	return rtcp.NackPair{PacketID: fsn, LostPackets: blp}
}
