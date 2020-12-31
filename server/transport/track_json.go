package transport

type TrackJSON struct {
	PayloadType uint8  `json:"payloadType"`
	SSRC        uint32 `json:"ssrc"`
	ID          string `json:"id"`
	Label       string `json:"label"`
}
