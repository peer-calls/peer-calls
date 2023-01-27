package pubsub

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var prometheusWebRTCTracksTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "webrtc_tracks_total",
	Help: "Total number of incoming webrtc tracks",
})

var prometheusWebRTCTracksActive = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "webrtc_tracks_active",
	Help: "Total number of incoming webrtc tracks",
})

var prometheusWebRTCTracksDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "webrtc_tracks_duration_seconds",
	Help:    "Duration of webrtc tracks",
	Buckets: []float64{1, 60, 5 * 60, 15 * 60, 30 * 60, 45 * 60, 60 * 60, 120 * 60},
})

var prometheusRTPPacketsReceived = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtp_packets_received2_total",
	Help: "Total number of received RTP packets",
})

var prometheusRTPPacketsReceivedBytes = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtp_packets_received2_bytes_total",
	Help: "Total number of received RTP bytes",
})

var prometheusRTPPacketsSent = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtp_packets_sent2_total",
	Help: "Total number of sent RTP packets",
})

var prometheusRTPPacketsSentBytes = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtp_packets_sent2_bytes_total",
	Help: "Total number of sent RTP bytes",
})
