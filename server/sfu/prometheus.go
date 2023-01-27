package sfu

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var prometheusRTCPPacketsReceived = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtcp_packets_received_total",
	Help: "Total number of received RTCP packets",
})

var prometheusRTCPPLIPacketsReceived = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtcp_pli_packets_received_total",
	Help: "Total number of received Picture Loss Indicator RTCP packets",
})

// var prometheusRTCPPacketsReceivedBytes = promauto.NewCounter(prometheus.CounterOpts{
// 	Name: "rtcp_packets_received2_bytes_total",
// 	Help: "Total number of received RTCP bytes",
// })

var prometheusRTCPPacketsSent = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rtcp_packets_sent2_total",
	Help: "Total number of sent RTCP packets",
})

// var prometheusRTCPPacketsSentBytes = promauto.NewGauge(prometheus.GaugeOpts{
// 	Name: "rtcp_packets_sent2_bytes_total",
// 	Help: "Total number of sent RTCP bytes",
// })
