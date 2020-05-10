package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var prometheusHomeViewsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "home_views_total",
	Help: "Total number of homepage views",
})

var prometheusCallJoinTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "call_join_total",
	Help: "Total number of new call requests",
})

var prometheusCallViewsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "call_views_total",
	Help: "Total number of homepage views",
})

var prometheusWSConnTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "ws_conn_total",
	Help: "Total number of opened websocket connections",
})

var prometheusWSConnActive = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "ws_conn_active",
	Help: "Total number of active websocket connections",
})

var prometheusWSConnErrTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "ws_conn_err_total",
	Help: "Total number of errored out websocket connections",
})

var prometheusWSConnDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "ws_conn_duration",
	Help: "Duration of websocket connections",
})

var prometheusWebRTCConnTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "webrtc_conn_total",
	Help: "Total number of opened webrtc connections",
})

var prometheusWebRTCConnActive = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "webrtc_conn_active",
	Help: "Total number of active webrtc connections",
})

// var prometheusWebRTCConnErrTotal = promauto.NewCounter(prometheus.CounterOpts{
// 	Name: "webrtc_conn_err_total",
// 	Help: "Total number of errored out webrtc connections",
// })

var prometheusWebRTCConnDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "webrtc_conn_duration_seconds",
	Help: "Duration of webrtc connections",
})

var prometheusWebRTCTracksTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "webrtc_tracks_total",
	Help: "Total number of incoming webrtc tracks",
})

var prometheusWebRTCTracksActive = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "webrtc_tracks_active",
	Help: "Total number of incoming webrtc tracks",
})

var prometheusWebRTCTracksDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "webrtc_tracks_duration_seconds",
	Help: "Duration of webrtc tracks",
})

var prometheusRTCPPacketsReceived = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rtcp_packets_received_total",
	Help: "Total number of received RTCP packets",
})

var prometheusRTCPPacketsSent = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rtcp_packets_sent_total",
	Help: "Total number of sent RTP packets",
})

var prometheusRTPPacketsReceived = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "rtp_packets_received_total",
	Help: "Total number of received RTP packets",
})
