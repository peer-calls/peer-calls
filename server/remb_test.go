package server

import (
	"testing"

	"github.com/pion/rtcp"
	"github.com/stretchr/testify/assert"
)

func newREMB(ssrc uint32, bitrate uint64) *rtcp.ReceiverEstimatedMaximumBitrate {
	return &rtcp.ReceiverEstimatedMaximumBitrate{
		SenderSSRC: ssrc,
		Bitrate:    bitrate,
	}
}

func TestTrackBitrateEstimators(t *testing.T) {
	assert := assert.New(t)

	var ssrc uint32 = 123
	b := NewTrackBitrateEstimators()

	assert.Equal(uint64(1000), b.Estimate("client1", newREMB(ssrc, 1000)).Bitrate)
	assert.Equal(uint64(900), b.Estimate("client2", newREMB(ssrc, 900)).Bitrate)
	assert.Equal(uint64(900), b.Estimate("client3", newREMB(ssrc, 1100)).Bitrate)

	assert.Equal(uint64(950), b.Estimate("client2", newREMB(ssrc, 950)).Bitrate)
	assert.Equal(uint64(1000), b.Estimate("client2", newREMB(ssrc, 1200)).Bitrate)

	b.RemoveReceiverEstimations("client1")
	assert.Equal(uint64(1100), b.Estimate("client2", newREMB(ssrc, 1300)).Bitrate)

	b.Remove(ssrc)

	assert.Equal(uint64(1500), b.Estimate("client4", newREMB(ssrc, 1500)).Bitrate)
}
