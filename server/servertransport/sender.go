package servertransport

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/transport/packetio"
)

type sender struct {
	buffer *packetio.Buffer
}

func newSender(buffer *packetio.Buffer) *sender {
	return &sender{
		buffer: buffer,
	}
}

var _ transport.Sender = &sender{}

func (s *sender) ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error) {
	b := make([]byte, ReceiveMTU)

	i, err := s.buffer.Read(b)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "reading RTCP")
	}

	packets, err := rtcp.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTCP")
	}

	// TODO interceptors

	return packets, nil, nil
}
