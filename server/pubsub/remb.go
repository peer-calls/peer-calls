package pubsub

import "github.com/peer-calls/peer-calls/v4/server/identifiers"

// BitrateEstimator estimates minimum, maximum and average bitrate. It is not
// safe for concurrent use.
type BitrateEstimator struct {
	min, max, avg float32

	needsMinMaxRecalc bool

	totalBitrate float32

	estimatesByClientID map[identifiers.ClientID]float32
}

// NewBitrateEstimator creates a new instance of BitrateEstimator.
func NewBitrateEstimator() *BitrateEstimator {
	return &BitrateEstimator{
		estimatesByClientID: map[identifiers.ClientID]float32{},
	}
}

// Feed records the estimated bitrate for client.
func (r *BitrateEstimator) Feed(clientID identifiers.ClientID, estimatedBitrate float32) {
	oldEstimatedBitrate, ok := r.estimatesByClientID[clientID]

	delete(r.estimatesByClientID, clientID)

	if ok && (oldEstimatedBitrate == r.min || oldEstimatedBitrate == r.max) {
		r.needsMinMaxRecalc = true
	}

	r.totalBitrate += -float32(oldEstimatedBitrate) + float32(estimatedBitrate)

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
	r.avg = r.totalBitrate / float32(len(r.estimatesByClientID))
}

func (r *BitrateEstimator) maybeRecalculateMinMax() {
	if r.needsMinMaxRecalc {
		r.recalculateMinMax()
	}
}

func (r *BitrateEstimator) recalculateMinMax() {
	var (
		min, max float32
		total    float32
	)

	for _, est := range r.estimatesByClientID {
		if min == 0 || est < min {
			min = est
		}

		if max == 0 || est > max {
			max = est
		}

		total += est
	}

	r.min = min
	r.max = max
	r.needsMinMaxRecalc = false
}

// Empty returns true when there are no estimations, false otherwise.
func (r *BitrateEstimator) Empty() bool {
	return len(r.estimatesByClientID) == 0
}

// Min returns the minimal bitrate.
func (r *BitrateEstimator) Min() float32 {
	r.maybeRecalculateMinMax()

	return r.min
}

// Max returns thet maximum bitrate.
func (r *BitrateEstimator) Max() float32 {
	r.maybeRecalculateMinMax()

	return r.max
}

// Avg returns the average bitrate.
func (r *BitrateEstimator) Avg() float32 {
	r.maybeRecalculateMinMax()

	return r.avg
}

func (r *BitrateEstimator) RemoveClientBitrate(clientID identifiers.ClientID) {
	oldEstimate, ok := r.estimatesByClientID[clientID]
	if !ok {
		return
	}

	delete(r.estimatesByClientID, clientID)
	r.totalBitrate -= float32(oldEstimate)

	if oldEstimate == r.min || oldEstimate == r.max {
		r.needsMinMaxRecalc = true
	}

	if size := len(r.estimatesByClientID); size > 0 {
		r.avg = r.totalBitrate / float32(size)
	} else {
		r.avg = 0
	}
}
