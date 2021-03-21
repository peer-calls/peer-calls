package pubsub

// BitrateEstimator estimates minimum, maximum and average bitrate. It is not
// safe for concurrent use.
type BitrateEstimator struct {
	min, max, avg uint64

	needsMinMaxRecalc bool

	totalBitrate float64

	estimatesByClientID map[string]uint64
}

// NewBitrateEstimator creates a new instance of BitrateEstimator.
func NewBitrateEstimator() *BitrateEstimator {
	return &BitrateEstimator{
		estimatesByClientID: map[string]uint64{},
	}
}

// Feed records the estimated bitrate for client.
func (r *BitrateEstimator) Feed(clientID string, estimatedBitrate uint64) {
	oldEstimatedBitrate, ok := r.estimatesByClientID[clientID]

	delete(r.estimatesByClientID, clientID)

	if ok && (oldEstimatedBitrate == r.min || oldEstimatedBitrate == r.max) {
		r.needsMinMaxRecalc = true
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

func (r *BitrateEstimator) maybeRecalculateMinMax() {
	if r.needsMinMaxRecalc {
		r.recalculateMinMax()
	}
}

func (r *BitrateEstimator) recalculateMinMax() {
	var (
		min, max uint64
		total    float64
	)

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
	r.needsMinMaxRecalc = false
}

// Empty returns true when there are no estimations, false otherwise.
func (r *BitrateEstimator) Empty() bool {
	return len(r.estimatesByClientID) == 0
}

// Min returns the minimal bitrate.
func (r *BitrateEstimator) Min() uint64 {
	r.maybeRecalculateMinMax()

	return r.min
}

// Max returns thet maximum bitrate.
func (r *BitrateEstimator) Max() uint64 {
	r.maybeRecalculateMinMax()

	return r.max
}

// Avg returns the average bitrate.
func (r *BitrateEstimator) Avg() uint64 {
	r.maybeRecalculateMinMax()

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
		r.needsMinMaxRecalc = true
	}

	if size := len(r.estimatesByClientID); size > 0 {
		r.avg = uint64(r.totalBitrate) / uint64(size)
	} else {
		r.avg = 0
	}
}
