package stats

import (
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const rtpSeqMod uint32 = 1 << 16

// Source contains per-Source state information. Implemented as per RFC 3550
// appendices A.1 and A.3.
type Source struct {
	// ssrc is the source SSRC.
	ssrc uint32
	// lastSenderReport is the NTP time from the latest sender report received
	// for this source.
	lastSenderReport NTPTime

	// maxSeq is the highes sequence number seen.
	maxSeq uint16
	// cycles is the shifted count of sequence number cycles
	cycles uint32
	// baseSeq is the base sequence number.
	baseSeq uint32
	// badSeq is the last 'bad' sequence number + 1.
	badSeq uint32
	// probation is sequential packets till source is valid.
	probation uint32
	// received counts the packets received.
	received uint32
	// expectedPrior is the packet expected at last interval.
	expectedPrior uint32
	// receivedPrior is the packet received at last interval.
	receivedPrior uint32
	// transit is the relative trans time for previous packet.
	transit uint32
	// jitter is the estimated jitter.
	jitter uint32
}

// NewSource creates a new instance of Source.
func (s *Source) NewSource(ssrc uint32) *Source {
	return &Source{
		ssrc: ssrc,
	}
}

// InitSeq is implemented according to the RFC 3550 Appendix A.1.
func (s *Source) InitSeq(seq uint16) {
	s.baseSeq = uint32(seq)
	s.maxSeq = seq
	// so seq == bad_seq is false.
	s.badSeq = rtpSeqMod + 1
	s.cycles = 0
	s.received = 0
	s.receivedPrior = 0
	s.expectedPrior = 0
}

// updateSeq is implemented according to the RFC 3550 Appendix A.1.
func (s *Source) updateSeq(seq uint16) bool {
	udelta := seq - s.maxSeq

	const (
		maxDropout    = 3000
		maxMisorder   = 100
		minSequential = 2
	)

	// Source is not valid until minSequential packets with sequential sequence
	// numbers have been received.
	switch {
	case s.probation != 0:
		// Packet is in sequence.
		if seq == s.maxSeq+1 {
			s.probation--
			s.maxSeq = seq
			if s.probation == 0 {
				s.InitSeq(seq)
				s.received++

				return true
			}
		} else {
			s.probation = minSequential - 1
			s.maxSeq = seq
		}

		return false
	case udelta < maxDropout:
		// In order, with permissible gap.
		if seq < s.maxSeq {
			// Sequence number wrapped - count another 64K cycle.
			s.cycles += rtpSeqMod
		}
		s.maxSeq = seq
	case udelta <= uint16(rtpSeqMod-maxMisorder):
		// The sequence number made a very large jump.
		if uint32(seq) == s.badSeq {
			// Two sequential packets: assume that the other side restarted without
			// telling us so just re-sync (i.e., pretend this was the first packet).
			s.InitSeq(seq)
		} else {
			s.badSeq = (uint32(seq) + 1) & (rtpSeqMod - 1)

			return false
		}
	default:
		// Duplicate or reordered packet.
	}

	s.received++

	return true
}

// Report is implemented according to the RFC 3550 Appendix A.8.
func (s *Source) updateJitter(packetTS, arrivalTS uint32) {
	if packetTS > arrivalTS {
		arrivalTS, packetTS = packetTS, arrivalTS
	}

	transit := arrivalTS - packetTS

	d := transit - s.transit
	s.transit = transit

	// See alternative below.
	// s.jitter += uint32(float64(1) / float64(16) * (float64(d) - float64(s.jitter)))

	// Alternatively, the jitter estimate can be kept as an integer, but
	// scaled to reduce round-off error.  The calculation is the same except
	// for the last line:
	s.jitter += d - ((s.jitter + 8) >> 4)
}

// Report is implemented according to the RFC 3550 Appendix A.3.
func (s *Source) ReceptionReport(now time.Time) rtcp.ReceptionReport {
	extendedMax := s.cycles + uint32(s.maxSeq)
	expected := extendedMax - s.baseSeq + 1

	// The number of packets lost is defined to be the number of packets expected
	// less the number of packets actually received.
	lost := expected - s.received

	// Since this signed number is carried in 24 bits, it should be clamped at
	// 0x7fffff for positive loss or 0x800000 for negative loss rather than
	// wrapping around.

	// The fraction of packets lost during the last reporting interval (since
	// the previous SR or RR packet was sent) is calculated from differences in
	// the expected and received packet counts across the interval, where
	// expected_prior and received_prior are the values saved when the previous
	// reception report was generated.
	expectedInterval := expected - s.expectedPrior
	s.expectedPrior = expected

	receivedInterval := s.received - s.receivedPrior
	s.receivedPrior = s.received

	lostInterval := expectedInterval - receivedInterval

	var fraction uint8

	if !(expectedInterval == 0 || lostInterval <= 0) {
		// The resulting fraction is an 8-bit fixed point number with the binary
		// point at the left edge.
		fraction = uint8((lostInterval << 8) / expectedInterval)
	}

	jitterShift := 4

	// From the RFC:
	//
	// Wallclock time (absolute date and time) is represented using the timestamp
	// format of the Network Time Protocol (NTP), which is in seconds relative to
	// 0h UTC on 1 January 1900 [4].  The full resolution NTP timestamp is a
	// 64-bit unsigned fixed-point number with the integer part in the first 32
	// bits and the fractional part in the last 32 bits. In some fields where a
	// more compact representation is appropriate, only the middle 32 bits are
	// used; that is, the low 16 bits of the integer part and the high 16 bits of
	// the fractional part.  The high 16 bits of the integer part must be
	// determined independently.
	lastSenderReport := s.lastSenderReport.Middle()

	var delay uint32

	if lastSenderReport > 0 {
		nowNTPMiddle := NewNTPTime(now).Middle()

		if lastSenderReport > nowNTPMiddle {
			nowNTPMiddle, lastSenderReport = lastSenderReport, nowNTPMiddle
		}

		// The delay, expressed in units of 1/65536 seconds, between
		// receiving the last SR packet from source SSRC_n and sending this
		// reception report block.  If no SR packet has been received yet
		// from SSRC_n, the DLSR field is set to zero.
		delay = nowNTPMiddle - lastSenderReport
	}

	return rtcp.ReceptionReport{
		Delay: delay,
		// Alternatively, the jitter estimate can be kept as an integer, but
		// scaled to reduce round-off error. (...)
		//
		// In this case, the estimate is sampled for the reception report as:
		Jitter:             s.jitter >> jitterShift,
		FractionLost:       fraction,
		LastSenderReport:   lastSenderReport,
		LastSequenceNumber: s.cycles + uint32(s.maxSeq),
		SSRC:               s.ssrc,
		TotalLost:          lost,
	}
}

func (s *Source) HandleSenderReport(r *rtcp.SenderReport) {
	s.lastSenderReport = NTPTime(r.NTPTime)
}

func (s *Source) HandleRTP(packet *rtp.Packet, now time.Time) {
	isValid := s.updateSeq(packet.SequenceNumber)
	_ = isValid // TODO

	s.updateJitter(packet.Timestamp, NewNTPTime(now).Middle())
}
