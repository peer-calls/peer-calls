module github.com/peer-calls/peer-calls

go 1.14

require (
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-redis/redis/v7 v7.2.0
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr v1.30.1
	github.com/google/uuid v1.1.1
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/pion/logging v0.2.2
	github.com/pion/rtcp v1.2.1
	github.com/pion/rtp v1.5.0
	github.com/pion/webrtc/v2 v2.2.11
	github.com/prometheus/client_golang v1.6.0
	github.com/stretchr/testify v1.5.1
	go.uber.org/goleak v1.0.0
	gopkg.in/yaml.v2 v2.2.8
	nhooyr.io/websocket v1.8.4
)

// replace github.com/pion/webrtc/v2 => github.com/jeremija/webrtc/v2 v2.2.6-0.20200420091005-4cc16a2df9e0
// replace github.com/pion/webrtc/v2 => ../../pion/webrtc
