package pubsub

import "fmt"

type BitrateEstimator struct {
	min, max, avg uint64

	totalBitrate float64

	estimatesByClientID map[string]uint64
}

func NewBitrateEstimator() *BitrateEstimator {
	return &BitrateEstimator{
		estimatesByClientID: map[string]uint64{},
	}
}

func (r *BitrateEstimator) Feed(clientID string, estimatedBitrate uint64) {
	oldEstimatedBitrate, ok := r.estimatesByClientID[clientID]

	delete(r.estimatesByClientID, clientID)

	if ok && (oldEstimatedBitrate == r.min || oldEstimatedBitrate == r.max) {
		r.recalculateMinMax()
	}

	r.totalBitrate += -float64(oldEstimatedBitrate) + float64(estimatedBitrate)

	r.estimatesByClientID[clientID] = estimatedBitrate

	if r.min == 0 || estimatedBitrate < r.min {
		r.min = estimatedBitrate
	}

	if r.max == 0 || estimatedBitrate > r.max {
		r.max = estimatedBitrate
	}

	r.recalculateAvg()
}

func (r *BitrateEstimator) recalculateAvg() {
	r.avg = uint64(r.totalBitrate) / uint64(len(r.estimatesByClientID))
}

func (r *BitrateEstimator) recalculateMinMax() {
	var (
		min, max uint64
		total    float64
	)

	fmt.Println("min max")

	for _, est := range r.estimatesByClientID {
		if min == 0 || est < min {
			min = est
		}

		if max == 0 || est > max {
			max = est
		}

		total += float64(est)
	}

	r.min = min
	r.max = max
}

func (r *BitrateEstimator) Empty() bool {
	return len(r.estimatesByClientID) == 0
}

func (r *BitrateEstimator) Min() uint64 {
	return r.min
}

func (r *BitrateEstimator) Max() uint64 {
	return r.max
}

func (r *BitrateEstimator) Avg() uint64 {
	return r.avg
}

func (r *BitrateEstimator) RemoveClientBitrate(clientID string) {
	oldEstimate, ok := r.estimatesByClientID[clientID]
	if !ok {
		return
	}

	delete(r.estimatesByClientID, clientID)
	r.totalBitrate -= float64(oldEstimate)

	if oldEstimate == r.min || oldEstimate == r.max {
		r.recalculateMinMax()
	}

	if size := len(r.estimatesByClientID); size > 0 {
		r.avg = uint64(r.totalBitrate) / uint64(size)
	} else {
		r.avg = 0
	}
}
