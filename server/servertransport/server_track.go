package servertransport

import (
	"sync/atomic"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/transport"
)

type ServerTrack struct {
	transport.SimpleTrack

	subCount int64

	onSub   func() error
	onUnsub func() error
}

func (s *ServerTrack) Sub() error {
	if subCount := atomic.AddInt64(&s.subCount, 1); subCount == 1 {
		return errors.Trace(s.onSub())
	}

	return nil
}

func (s *ServerTrack) Unsub() error {
	if subCount := atomic.AddInt64(&s.subCount, -1); subCount == 0 {
		return errors.Trace(s.onUnsub())
	}

	return nil
}
