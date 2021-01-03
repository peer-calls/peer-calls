package udptransport2

import "github.com/peer-calls/peer-calls/server/servertransport"

type Transport struct {
	*servertransport.Transport
	streamID string
}

func (t Transport) StreamID() string {
	return t.streamID
}
