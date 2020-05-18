package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackBitrateEstimators(t *testing.T) {
	assert := assert.New(t)

	ssrcs := []uint32{123}
	b := NewTrackBitrateEstimators()

	assert.Equal(uint64(1000), b.Estimate("client1", ssrcs, 1000))
	assert.Equal(uint64(900), b.Estimate("client2", ssrcs, 900))
	assert.Equal(uint64(900), b.Estimate("client3", ssrcs, 1100))

	assert.Equal(uint64(950), b.Estimate("client2", ssrcs, 950))
	assert.Equal(uint64(1000), b.Estimate("client2", ssrcs, 1200))

	b.RemoveReceiverEstimations("client1")
	assert.Equal(uint64(1100), b.Estimate("client2", ssrcs, 1300))

	b.Remove(ssrcs[0])

	assert.Equal(uint64(1500), b.Estimate("client4", ssrcs, 1500))
}
