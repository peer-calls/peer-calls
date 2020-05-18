package server

import (
	"math"
	"sync"
)

type BitrateEstimator struct {
	bitrate   uint64
	estimates map[string]uint64
}

func NewBitrateEstimator() *BitrateEstimator {
	return &BitrateEstimator{
		bitrate:   math.MaxUint64,
		estimates: map[string]uint64{},
	}
}

func (r *BitrateEstimator) Estimate(receiverClientID string, bitrate uint64) uint64 {
	r.estimates[receiverClientID] = bitrate

	if bitrate <= r.bitrate {
		r.bitrate = bitrate
		return bitrate
	}

	minBitrate := bitrate
	for clientID, bitrate := range r.estimates {
		if clientID != receiverClientID && bitrate < minBitrate {
			minBitrate = bitrate
		}
	}
	r.bitrate = minBitrate

	return minBitrate
}

func (r *BitrateEstimator) RemoveEstimation(receiverClientID string) {
	delete(r.estimates, receiverClientID)
}

type TrackBitrateEstimators struct {
	mu         sync.Mutex
	estimators map[uint32]*BitrateEstimator
}

func NewTrackBitrateEstimators() *TrackBitrateEstimators {
	return &TrackBitrateEstimators{
		estimators: map[uint32]*BitrateEstimator{},
	}
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (r *TrackBitrateEstimators) Estimate(
	receiverClientID string,
	ssrcs []uint32,
	bitrate uint64,
) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	var minBitrate uint64 = math.MaxUint64
	for _, ssrc := range ssrcs {
		estimator, ok := r.estimators[ssrc]
		if !ok {
			estimator = NewBitrateEstimator()
			r.estimators[ssrc] = estimator
		}
		minBitrate = min(minBitrate, estimator.Estimate(receiverClientID, bitrate))
	}

	return minBitrate
}

func (r *TrackBitrateEstimators) RemoveReceiverEstimations(
	receiverClientID string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, estimator := range r.estimators {
		estimator.RemoveEstimation(receiverClientID)
	}
}

func (r *TrackBitrateEstimators) Remove(ssrc uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.estimators, ssrc)
}
