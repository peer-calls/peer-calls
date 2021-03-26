package servertransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/atomic"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/transport/packetio"
)

type sender struct {
	buffer *packetio.Buffer

	closed *atomic.Bool

	interceptor           interceptor.Interceptor
	interceptorRTCPReader interceptor.RTCPReader
}

func newSender(buffer *packetio.Buffer, i interceptor.Interceptor, closed *atomic.Bool) *sender {
	s := &sender{
		buffer:      buffer,
		interceptor: i,
		closed:      closed,
	}

	s.interceptorRTCPReader = i.BindRTCPReader(interceptor.RTCPReaderFunc(s.read))

	return s
}

var _ transport.Sender = &sender{}

func (s *sender) ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error) {
	b := make([]byte, ReceiveMTU)

	i, a, err := s.interceptorRTCPReader.Read(b, interceptor.Attributes{})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "reading RTCP")
	}

	packets, err := rtcp.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTCP")
	}

	return packets, a, nil
}

func (s *sender) read(in []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
	i, err := s.buffer.Read(in)

	return i, a, errors.Trace(err)
}

func (s *sender) Close() {
	s.closed.Set(true)
}
