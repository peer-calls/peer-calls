package udptransport2

import "github.com/peer-calls/peer-calls/server/servertransport"

type Transport struct {
	*servertransport.Transport
	StreamID string
}
