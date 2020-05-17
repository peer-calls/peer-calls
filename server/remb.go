package server

import (
	"math"
	"sync"

	"github.com/pion/rtcp"
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

func (r *BitrateEstimator) Estimate(
	receiverClientID string,
	remb *rtcp.ReceiverEstimatedMaximumBitrate,
) *rtcp.ReceiverEstimatedMaximumBitrate {

	r.estimates[receiverClientID] = remb.Bitrate

	if remb.Bitrate <= r.bitrate {
		r.bitrate = remb.Bitrate
		return remb
	}

	minBitrate := remb.Bitrate
	for clientID, bitrate := range r.estimates {
		if clientID != receiverClientID && bitrate < minBitrate {
			minBitrate = bitrate
		}
	}
	r.bitrate = minBitrate

	return &rtcp.ReceiverEstimatedMaximumBitrate{
		SenderSSRC: remb.SenderSSRC,
		Bitrate:    minBitrate,
		SSRCs:      remb.SSRCs,
	}
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

func (r *TrackBitrateEstimators) Estimate(
	receiverClientID string,
	remb *rtcp.ReceiverEstimatedMaximumBitrate,
) *rtcp.ReceiverEstimatedMaximumBitrate {
	r.mu.Lock()
	defer r.mu.Unlock()

	estimator, ok := r.estimators[remb.SenderSSRC]
	if !ok {
		estimator = NewBitrateEstimator()
		r.estimators[remb.SenderSSRC] = estimator
	}

	return estimator.Estimate(receiverClientID, remb)
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
