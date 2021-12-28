package servertransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/transport/packetio"
)

type rtcpReader struct {
	buffer *packetio.Buffer

	interceptor           interceptor.Interceptor
	interceptorRTCPReader interceptor.RTCPReader
}

var _ transport.RTCPReader = &rtcpReader{}

func newRTCPReader(buffer *packetio.Buffer, i interceptor.Interceptor) *rtcpReader {
	s := &rtcpReader{
		buffer:      buffer,
		interceptor: i,
	}

	s.interceptorRTCPReader = i.BindRTCPReader(interceptor.RTCPReaderFunc(s.read))

	return s
}

func (s *rtcpReader) ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error) {
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

func (s *rtcpReader) read(in []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
	i, err := s.buffer.Read(in)

	return i, a, errors.Trace(err)
}

func (s *rtcpReader) Close() {
	// TODO no way to unbind RTCPReader.
}
