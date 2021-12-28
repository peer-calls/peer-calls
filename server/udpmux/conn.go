package udpmux

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

type Conn interface {
	net.Conn
	Done() <-chan struct{}
}

type conn struct {
	logger logger.Logger

	conn  net.PacketConn
	laddr net.Addr
	raddr net.Addr

	readChan             chan []byte
	closeConnRequestChan chan closeConnRequest
	torndown             chan struct{}
}

var _ Conn = &conn{}

func (m *conn) Close() error {
	req := closeConnRequest{
		conn:    m,
		errChan: make(chan error, 1),
	}

	select {
	case m.closeConnRequestChan <- req:
	case <-m.torndown:
		return nil
	}

	err := <-req.errChan

	return errors.Trace(err)
}

func (m *conn) close() {
	close(m.readChan)
	close(m.torndown)
}

func (m *conn) Done() <-chan struct{} {
	return m.torndown
}

func (m *conn) Read(b []byte) (int, error) {
	buf, ok := <-m.readChan
	if !ok {
		return 0, errors.Annotatef(io.ErrClosedPipe, "raddr: %s", m.raddr)
	}

	copy(b, buf)
	m.logger.Trace("recv", logger.Ctx{
		"data": buf,
	})

	return len(buf), nil
}

func (m *conn) Write(b []byte) (int, error) {
	select {
	case <-m.torndown:
		return 0, errors.Annotatef(io.ErrClosedPipe, "raddr: %s", m.raddr)
	default:
		m.logger.Trace("send", logger.Ctx{
			"data": b,
		})

		i, err := m.conn.WriteTo(b, m.raddr)

		return i, errors.Annotate(err, "write")
	}
}

func (m *conn) LocalAddr() net.Addr {
	return m.laddr
}

func (m *conn) RemoteAddr() net.Addr {
	return m.raddr
}

func (m *conn) SetDeadline(t time.Time) error {
	// TODO
	return nil
}

func (m *conn) SetReadDeadline(t time.Time) error {
	// TODO
	return nil
}

func (m *conn) SetWriteDeadline(t time.Time) error {
	// TODO
	return nil
}

func (m *conn) String() string {
	if s, ok := m.conn.(stringer); ok {
		return fmt.Sprintf("%s [%s %s]", s.String(), m.laddr, m.raddr)
	}

	return fmt.Sprintf("[%s %s]", m.laddr, m.raddr)
}

type stringer interface {
	String() string
}
