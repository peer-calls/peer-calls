package udptransport2

import (
	"net"
	"time"

	"github.com/juju/errors"
	"github.com/pion/sctp"
)

// streamConn wraps the sctp.Stream inta net.Conn to make it easier to use with
// stringmux.
type streamConn struct {
	*sctp.Stream
	laddr net.Addr
	raddr net.Addr
}

func newStreamConn(stream *sctp.Stream, laddr net.Addr, raddr net.Addr) *streamConn {
	return &streamConn{
		Stream: stream,
		laddr:  laddr,
		raddr:  raddr,
	}
}

var _ net.Conn = &streamConn{}

func (s *streamConn) LocalAddr() net.Addr {
	return s.laddr
}

func (s *streamConn) RemoteAddr() net.Addr {
	return s.raddr
}

func (s *streamConn) SetDeadline(t time.Time) error {
	return errors.Errorf("not implemented")
}

func (s *streamConn) SetWriteDeadline(t time.Time) error {
	return errors.Errorf("not implemented")
}

func (s *streamConn) SetReadDeadline(t time.Time) error {
	return errors.Errorf("not implemented")
}
