package server

import (
	"fmt"
	"testing"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffer_snDelta(t *testing.T) {
	assert := assert.New(t)
	assert.EqualValues(5, snDelta(5, 10))

	assert.EqualValues(1, snDelta(maxSN, 0))
	assert.EqualValues(2, snDelta(maxSN-1, 0))
	assert.EqualValues(3, snDelta(maxSN-2, 0))
	assert.EqualValues(4, snDelta(maxSN-3, 0))

	assert.EqualValues(2, snDelta(maxSN, 1))
	assert.EqualValues(3, snDelta(maxSN, 2))
	assert.EqualValues(4, snDelta(maxSN, 3))
}

func TestBuffer_tsDelta(t *testing.T) {
	assert := assert.New(t)
	assert.EqualValues(2, tsDelta(4, 2))
	assert.EqualValues(3, tsDelta(5, 2))
	assert.EqualValues(2, tsDelta(2, 4))
	assert.EqualValues(3, tsDelta(2, 5))
}

func TestBuffer_Push_FirstPacket(t *testing.T) {
	assert := assert.New(t)
	b := NewBuffer()
	p := rtp.Packet{}
	p.SequenceNumber = 123
	p.Timestamp = 456
	p.SSRC = 789

	rtcpPkt := b.Push(&p)
	assert.Nil(rtcpPkt, "Unexpected rtcp packet")
	assert.EqualValues(123, b.lastPushSN)
	assert.EqualValues(123, b.lastNackSN)
	assert.EqualValues(123-1, b.lastClearSN)
	assert.EqualValues(456, b.lastClearTS)
	assert.EqualValues(789, b.SSRC())
}

func TestBuffer_Push_Nack_None(t *testing.T) {
	assert := assert.New(t)
	b := NewBuffer()

	for i := uint16(0); i < maxNackPairSize+1; i++ {
		p := rtp.Packet{}
		p.Timestamp = 1
		p.SequenceNumber = i
		assert.Nil(b.Push(&p), "unexpected rtcp packet")
	}
}

func containsUint16(slice []uint16, value uint16) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func TestBuffer_Push_NackPair_Single(t *testing.T) {
	assert := assert.New(t)

	for _, test := range []struct {
		start        uint16
		end          uint16
		drop         []uint16
		expectedNack []uint16
	}{
		{0, maxNackPairSize, []uint16{15}, []uint16{15}},
		{0, maxNackPairSize, []uint16{14, 15}, []uint16{14, 15}},
		{0, maxNackPairSize, []uint16{1, 2}, []uint16{1, 2}},
		{maxSN, maxNackPairSize - 1, []uint16{1, 2}, []uint16{1, 2}},
		{maxSN - 1, maxNackPairSize - 2, []uint16{1, 2}, []uint16{1, 2}},
		{maxSN - 2, maxNackPairSize - 3, []uint16{1, 2}, []uint16{1, 2}},
	} {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			b := NewBuffer()

			var ssrc uint32 = 111
			start := test.start
			end := test.end
			drop := test.drop
			expectedNack := test.expectedNack

			for i := start; i != end; i++ {
				if containsUint16(drop, i) {
					continue
				}
				p := rtp.Packet{}
				p.Timestamp = 1
				p.SequenceNumber = i
				p.SSRC = ssrc
				assert.Nil(b.Push(&p), "unexpected rtcp packet")
			}

			p := rtp.Packet{}
			p.Timestamp = 1
			p.SequenceNumber = end
			p.SSRC = ssrc
			rtcpPkt := b.Push(&p)

			assert.NotNil(rtcpPkt, "expected a rtcp packet")
			nackPkt, ok := rtcpPkt.(*rtcp.TransportLayerNack)
			require.True(t, ok, "expected a TransportLayerNack packet")
			assert.Equal(ssrc, nackPkt.SenderSSRC)
			assert.Equal(ssrc, nackPkt.MediaSSRC)
			require.Equal(t, 1, len(nackPkt.Nacks))

			nackPair := nackPkt.Nacks[0]
			assert.Equal(expectedNack, nackPair.PacketList(), "expected NACK packet(s)")
		})
	}
}

func TestBuffer_Push_NackPair_IrregularNackWindowSize(t *testing.T) {
	assert := assert.New(t)
	b := NewBuffer()
	b.nackWindowSize = maxNackPairSize + 1

	var ssrc uint32 = 111
	start := maxSN - 2
	end := start + b.nackWindowSize
	drop := []uint16{maxSN, 0, end - 1}

	for i := start; i != end; i++ {
		if containsUint16(drop, i) {
			continue
		}
		p := rtp.Packet{}
		p.Timestamp = 1
		p.SequenceNumber = i
		p.SSRC = ssrc
		assert.Nil(b.Push(&p), "unexpected rtcp packet")
	}

	p := rtp.Packet{}
	p.Timestamp = 1
	p.SequenceNumber = end
	p.SSRC = ssrc
	rtcpPkt := b.Push(&p)

	assert.NotNil(rtcpPkt, "expected a rtcp packet")
	nackPkt, ok := rtcpPkt.(*rtcp.TransportLayerNack)
	require.True(t, ok, "expected a TransportLayerNack packet")
	assert.Equal(ssrc, nackPkt.SenderSSRC)
	assert.Equal(ssrc, nackPkt.MediaSSRC)
	require.Equal(t, 2, len(nackPkt.Nacks))

	assert.Equal([]uint16{maxSN, 0}, nackPkt.Nacks[0].PacketList(), "expected NACK packet(s)")
	assert.Equal([]uint16{end - 1}, nackPkt.Nacks[1].PacketList(), "expected NACK packet(s)")
}

func TestBuffer_Push_ClearOldPackets(t *testing.T) {
	assert := assert.New(t)
	b := NewBuffer()

	for i := uint16(0); i < 5; i++ {
		p := rtp.Packet{}
		if i == 2 {
			continue
		}
		p.Timestamp = uint32(i) * videoClock
		p.SequenceNumber = i
		assert.Nil(b.Push(&p), "unexpected rtcp packet")
	}

	assert.Nil(b.GetPacket(0))
	assert.Nil(b.GetPacket(1))
	assert.Nil(b.GetPacket(2))
	assert.NotNil(b.GetPacket(3))
	assert.NotNil(b.GetPacket(4))
}
