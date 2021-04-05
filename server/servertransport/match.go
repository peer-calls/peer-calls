package servertransport

import (
	"bytes"
	"encoding/binary"
)

// Functions borrowed from pion/ion
// Also see https://tools.ietf.org/html/rfc7983 for more info

// MatchFunc allows custom logic for mapping packets to an Endpoint
type MatchFunc func([]byte) bool

// MatchRange is a MatchFunc that accepts packets with the first byte in [lower..upper]
func MatchRange(lower byte, upper byte) MatchFunc {
	return func(buf []byte) bool {
		if len(buf) == 0 {
			return false
		}
		b := buf[0]
		return b >= lower && b <= upper
	}
}

// MatchRTPOrRTCP is a MatchFunc that accepts packets with the first byte in [128..191]
// as defied in RFC7983
func MatchRTPOrRTCP(b []byte) bool {
	return MatchRange(128, 191)(b)
}

func isRTCP(buf []byte) bool {
	// Not long enough to determine RTP/RTCP
	if len(buf) < 4 {
		return false
	}

	var rtcpPacketType uint8
	r := bytes.NewReader([]byte{buf[1]})
	if err := binary.Read(r, binary.BigEndian, &rtcpPacketType); err != nil {
		return false
	} else if rtcpPacketType >= 192 && rtcpPacketType <= 223 {
		return true
	}

	return false
}

// MatchRTP is a MatchFunc that only matches SRTP and not SRTCP
func MatchRTP(buf []byte) bool {
	return MatchRTPOrRTCP(buf) && !isRTCP(buf)
}

// MatchSRTCP is a MatchFunc that only matches SRTCP and not SRTP
func MatchRTCP(buf []byte) bool {
	return MatchRTPOrRTCP(buf) && isRTCP(buf)
}
